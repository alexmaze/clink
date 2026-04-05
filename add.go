package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexmaze/clink/config"
	"github.com/alexmaze/clink/lib/color"
	"github.com/alexmaze/clink/lib/fileutil"
	"github.com/alexmaze/clink/lib/icon"
	"github.com/alexmaze/clink/lib/spinner"
	"github.com/alexmaze/clink/lib/tui"
	"gopkg.in/yaml.v3"
)

var errAddCancelled = errors.New("add canceled")

type AddOpts struct {
	ConfigPath string
	Mode       string
	Dest       string
	Rule       string
	Name       string
	Yes        bool
	Source     string
}

type addSpec struct {
	ConfigPath       string
	ConfigDir        string
	SourceAbs        string
	ManagedSourceAbs string
	ManagedSourceRel string
	Dest             string
	RuleName         string
	Mode             config.Mode
	AppendToRule     bool
}

type addContext struct {
	doc        *yaml.Node
	configFile *config.ConfigFile
	configPath string
	configDir  string
	sourceAbs  string
	destAbs    string
	pathType   fileutil.PathType
}

func parseAddArgs(args []string) (AddOpts, error) {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var opts AddOpts
	fs.StringVar(&opts.ConfigPath, "c", "", "specify config file path")
	fs.StringVar(&opts.ConfigPath, "config", "", "specify config file path")
	fs.StringVar(&opts.Mode, "mode", "", "specify mode: symlink or copy")
	fs.StringVar(&opts.Dest, "dest", "", "specify destination path")
	fs.StringVar(&opts.Rule, "rule", "", "append to an existing rule")
	fs.StringVar(&opts.Name, "name", "", "specify new rule name")
	fs.BoolVar(&opts.Yes, "y", false, "accept defaults without prompt")
	fs.BoolVar(&opts.Yes, "yes", false, "accept defaults without prompt")

	if err := fs.Parse(args); err != nil {
		return AddOpts{}, err
	}
	if opts.ConfigPath == "" {
		return AddOpts{}, fmt.Errorf("`clink add` requires -c or --config")
	}

	rest := fs.Args()
	if len(rest) != 1 {
		return AddOpts{}, fmt.Errorf("usage: clink add <source> -c <config.yaml>")
	}
	if opts.Rule != "" && opts.Name != "" {
		return AddOpts{}, fmt.Errorf("--rule and --name cannot be used together")
	}
	opts.Source = rest[0]
	return opts, nil
}

func runAdd(opts AddOpts) error {
	sp := spinner.New()

	ctx, err := prepareAddContext(sp, opts)
	if err != nil {
		return err
	}

	spec, err := buildAddSpec(ctx, opts)
	if err != nil {
		return err
	}

	if err := ensureUniqueDest(ctx.configFile, spec.Dest); err != nil {
		return err
	}

	if err := ensureManagedSourceAllowed(ctx.configFile, spec.SourceAbs); err != nil {
		return err
	}

	if !opts.Yes {
		if !confirmAdd(spec) {
			return errAddCancelled
		}
	}

	if spec.SourceAbs != spec.ManagedSourceAbs {
		if err := importSourceToManagedPath(spec.SourceAbs, spec.ManagedSourceAbs); err != nil {
			return err
		}
		sp.CheckPoint(icon.IconCheck, color.ColorGreen,
			fmt.Sprintf("Imported source to %s", spec.ManagedSourceAbs),
			color.ColorReset)
	} else {
		sp.CheckPoint(icon.IconInfo, color.ColorCyan,
			fmt.Sprintf("Source already under config dir: %s", spec.ManagedSourceRel),
			color.ColorReset)
	}

	if err := applyAddSpec(ctx.doc, ctx.configFile, spec); err != nil {
		return err
	}
	if err := writeYAMLDocument(spec.ConfigPath, ctx.doc); err != nil {
		return err
	}

	sp.Success("Rule updated")
	sp.CheckPoint(icon.IconInfo, color.ColorCyan,
		fmt.Sprintf("Config: %s", spec.ConfigPath), color.ColorReset)
	sp.CheckPoint(icon.IconInfo, color.ColorCyan,
		fmt.Sprintf("Run: clink -c %s", spec.ConfigPath), color.ColorReset)
	return nil
}

func prepareAddContext(sp spinner.Spinner, opts AddOpts) (*addContext, error) {
	absConfigPath, err := filepath.Abs(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config path %v: %w", opts.ConfigPath, err)
	}
	if _, err := os.Stat(absConfigPath); err != nil {
		return nil, fmt.Errorf("config file not found: %s", absConfigPath)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	sourceAbs, err := fileutil.ParsePath(cwd, opts.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source path %v: %w", opts.Source, err)
	}
	if sourceAbs == absConfigPath {
		return nil, fmt.Errorf("source cannot be the config file itself")
	}

	exists, err := fileutil.IsFileExists(sourceAbs)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("source file/folder do not exist %v", sourceAbs)
	}

	pathType, err := fileutil.GetPathType(sourceAbs)
	if err != nil {
		return nil, err
	}

	doc, configFile, err := loadConfigDocument(absConfigPath)
	if err != nil {
		return nil, err
	}

	sp.CheckPoint(icon.IconInfo, color.ColorCyan, "Using: "+absConfigPath, color.ColorReset)
	sp.CheckPoint(icon.IconInfo, color.ColorCyan, "Source: "+sourceAbs, color.ColorReset)

	return &addContext{
		doc:        doc,
		configFile: configFile,
		configPath: absConfigPath,
		configDir:  filepath.Dir(absConfigPath),
		sourceAbs:  sourceAbs,
		pathType:   pathType,
	}, nil
}

func buildAddSpec(ctx *addContext, opts AddOpts) (*addSpec, error) {
	destAbs, err := resolveAddDest(ctx.configDir, ctx.sourceAbs, opts.Dest)
	if err != nil {
		return nil, err
	}

	appendToRule, ruleName, mode, err := chooseRuleAndMode(ctx, opts)
	if err != nil {
		return nil, err
	}

	if !opts.Yes {
		appendToRule, ruleName, mode, destAbs, err = promptAddChoices(ctx, appendToRule, ruleName, mode, destAbs)
		if err != nil {
			return nil, err
		}
	}

	managedRel, managedAbs, err := planManagedSourcePath(ctx.configDir, ctx.sourceAbs, ruleName, ctx.pathType)
	if err != nil {
		return nil, err
	}

	return &addSpec{
		ConfigPath:       ctx.configPath,
		ConfigDir:        ctx.configDir,
		SourceAbs:        ctx.sourceAbs,
		ManagedSourceAbs: managedAbs,
		ManagedSourceRel: managedRel,
		Dest:             destAbs,
		RuleName:         ruleName,
		Mode:             mode,
		AppendToRule:     appendToRule,
	}, nil
}

func resolveAddDest(configDir, sourceAbs, rawDest string) (string, error) {
	dest := rawDest
	if dest == "" {
		dest = sourceAbs
	}
	if filepath.IsAbs(dest) || strings.HasPrefix(dest, "~/") {
		return fileutil.ParsePath(configDir, dest)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	return fileutil.ParsePath(cwd, dest)
}

func chooseRuleAndMode(ctx *addContext, opts AddOpts) (bool, string, config.Mode, error) {
	if opts.Mode != "" {
		if opts.Mode != string(config.ModeSymlink) && opts.Mode != string(config.ModeCopy) {
			return false, "", "", fmt.Errorf("unsupported add mode %q, only symlink/copy are allowed", opts.Mode)
		}
	}

	if opts.Rule != "" {
		rule := findRuleByName(ctx.configFile, opts.Rule)
		if rule == nil {
			return false, "", "", fmt.Errorf("rule not found: %q", opts.Rule)
		}
		if rule.Mode == config.ModeSSH {
			return false, "", "", fmt.Errorf("rule %q uses ssh mode, `clink add` does not support appending to ssh rules", rule.Name)
		}
		return true, rule.Name, rule.Mode, nil
	}

	ruleName := opts.Name
	if ruleName == "" {
		ruleName = uniqueRuleName(ctx.configFile, filepath.Base(ctx.sourceAbs))
	}

	mode := inferDefaultAddMode(ctx.configFile)
	if opts.Mode != "" {
		mode = config.Mode(opts.Mode)
	}

	return false, ruleName, mode, nil
}

func promptAddChoices(ctx *addContext, appendToRule bool, ruleName string, mode config.Mode, destAbs string) (bool, string, config.Mode, string, error) {
	localRules := listLocalRules(ctx.configFile)
	if len(localRules) > 0 {
		items := []string{
			"新建 rule",
			"追加到已有 rule",
		}
		idx, err := tui.RunSelect("选择 rule 归属", items)
		if err != nil {
			return false, "", "", "", errAddCancelled
		}
		appendToRule = idx == 1
	}

	if appendToRule {
		names := make([]string, 0, len(localRules))
		for _, rule := range localRules {
			names = append(names, fmt.Sprintf("%s  [%s]", rule.Name, rule.Mode))
		}
		idx, err := tui.RunSelect("选择要追加的 rule", names)
		if err != nil {
			return false, "", "", "", errAddCancelled
		}
		ruleName = localRules[idx].Name
		mode = localRules[idx].Mode
	} else {
		nameInput, err := tui.RunInput("输入 rule 名称", ruleName)
		if err != nil {
			return false, "", "", "", errAddCancelled
		}
		nameInput = strings.TrimSpace(nameInput)
		if nameInput == "" {
			return false, "", "", "", fmt.Errorf("rule name cannot be empty")
		}
		if existing := findRuleByName(ctx.configFile, nameInput); existing != nil {
			return false, "", "", "", fmt.Errorf("rule %q already exists", nameInput)
		}
		ruleName = nameInput

		modeOptions := []config.Mode{mode}
		if mode == config.ModeSymlink {
			modeOptions = append(modeOptions, config.ModeCopy)
		} else {
			modeOptions = append(modeOptions, config.ModeSymlink)
		}

		labels := make([]string, 0, len(modeOptions))
		for _, candidate := range modeOptions {
			labels = append(labels, string(candidate))
		}
		idx, err := tui.RunSelect("选择分发模式", labels)
		if err != nil {
			return false, "", "", "", errAddCancelled
		}
		mode = modeOptions[idx]
	}

	destInput, err := tui.RunInput("确认目标路径 dest", destAbs)
	if err != nil {
		return false, "", "", "", errAddCancelled
	}
	destInput = strings.TrimSpace(destInput)
	if destInput == "" {
		return false, "", "", "", fmt.Errorf("destination cannot be empty")
	}
	destAbs, err = resolveAddDest(ctx.configDir, ctx.sourceAbs, destInput)
	if err != nil {
		return false, "", "", "", err
	}

	return appendToRule, ruleName, mode, destAbs, nil
}

func confirmAdd(spec *addSpec) bool {
	return tui.RunConfirm(tui.ConfirmOpts{
		Title: "即将把文件纳入 clink 管理",
		Bullets: []string{
			fmt.Sprintf("配置文件: %s", spec.ConfigPath),
			fmt.Sprintf("rule: %s", spec.RuleName),
			fmt.Sprintf("mode: %s", spec.Mode),
			fmt.Sprintf("源文件: %s", spec.SourceAbs),
			fmt.Sprintf("托管路径: %s", spec.ManagedSourceRel),
			fmt.Sprintf("目标路径: %s", spec.Dest),
		},
	})
}

func loadConfigDocument(path string) (*yaml.Node, *config.ConfigFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config file %v: %w", path, err)
	}

	var doc yaml.Node
	if len(strings.TrimSpace(string(raw))) == 0 {
		doc = yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				newMapNode(),
			},
		}
	} else if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, nil, fmt.Errorf("failed to parse yaml document %v: %w", path, err)
	}

	configFile, err := config.ParseConfigFileOnly(path)
	if err != nil {
		return nil, nil, err
	}
	if configFile == nil {
		configFile = &config.ConfigFile{}
	}
	return &doc, configFile, nil
}

func inferDefaultAddMode(configFile *config.ConfigFile) config.Mode {
	switch configFile.Mode {
	case config.ModeCopy:
		return config.ModeCopy
	case config.ModeSymlink:
		return config.ModeSymlink
	default:
		return config.ModeSymlink
	}
}

func findRuleByName(configFile *config.ConfigFile, name string) *config.Rule {
	for _, rule := range configFile.Rules {
		if strings.EqualFold(rule.Name, name) {
			return rule
		}
	}
	return nil
}

func listLocalRules(configFile *config.ConfigFile) []*config.Rule {
	rules := make([]*config.Rule, 0, len(configFile.Rules))
	for _, rule := range configFile.Rules {
		if rule.Mode == config.ModeSSH {
			continue
		}
		rules = append(rules, rule)
	}
	return rules
}

func uniqueRuleName(configFile *config.ConfigFile, base string) string {
	name := strings.TrimSpace(base)
	if name == "" {
		name = "new rule"
	}
	candidate := name
	for idx := 2; findRuleByName(configFile, candidate) != nil; idx++ {
		candidate = fmt.Sprintf("%s-%d", name, idx)
	}
	return candidate
}

func planManagedSourcePath(configDir, sourceAbs, ruleName string, pathType fileutil.PathType) (string, string, error) {
	rel, inside, err := relativeToConfigDir(configDir, sourceAbs)
	if err != nil {
		return "", "", err
	}
	if inside {
		return rel, sourceAbs, nil
	}

	slug := slugify(ruleName)
	if slug == "" {
		slug = "rule"
	}
	base := filepath.Base(sourceAbs)
	targetDir := filepath.Join(configDir, ".src", slug)
	targetPath, err := uniqueManagedPath(targetDir, base, pathType)
	if err != nil {
		return "", "", err
	}
	rel, err = formatRelativeToConfigDir(configDir, targetPath)
	if err != nil {
		return "", "", err
	}
	return rel, targetPath, nil
}

func relativeToConfigDir(configDir, path string) (string, bool, error) {
	cleanConfigDir := filepath.Clean(configDir)
	cleanPath := filepath.Clean(path)
	rel, err := filepath.Rel(cleanConfigDir, cleanPath)
	if err != nil {
		return "", false, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", false, nil
	}
	return normalizeRelativePath(rel), true, nil
}

func formatRelativeToConfigDir(configDir, path string) (string, error) {
	rel, err := filepath.Rel(configDir, path)
	if err != nil {
		return "", err
	}
	return normalizeRelativePath(rel), nil
}

func normalizeRelativePath(rel string) string {
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return "./."
	}
	if strings.HasPrefix(rel, "./") {
		return rel
	}
	return "./" + rel
}

func uniqueManagedPath(targetDir, base string, pathType fileutil.PathType) (string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create managed source dir %s: %w", targetDir, err)
	}

	ext := filepath.Ext(base)
	nameOnly := strings.TrimSuffix(base, ext)
	if pathType == fileutil.PathTypeFolder {
		ext = ""
		nameOnly = base
	}
	if nameOnly == "" {
		nameOnly = "item"
	}

	candidate := filepath.Join(targetDir, base)
	for idx := 2; ; idx++ {
		exists, err := destExists(candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		nextBase := fmt.Sprintf("%s-%d%s", nameOnly, idx, ext)
		candidate = filepath.Join(targetDir, nextBase)
	}
}

func importSourceToManagedPath(sourceAbs, managedAbs string) error {
	if err := os.MkdirAll(filepath.Dir(managedAbs), 0755); err != nil {
		return fmt.Errorf("failed to create managed source parent: %w", err)
	}
	return copyPath(sourceAbs, managedAbs)
}

func ensureUniqueDest(configFile *config.ConfigFile, dest string) error {
	for _, rule := range configFile.Rules {
		for _, item := range rule.Items {
			if item.Destination == dest {
				return fmt.Errorf("destination already exists in config: %s", dest)
			}
		}
	}
	return nil
}

func ensureManagedSourceAllowed(configFile *config.ConfigFile, sourceAbs string) error {
	for _, rule := range configFile.Rules {
		for _, item := range rule.Items {
			if item.Source == sourceAbs {
				return fmt.Errorf("source is already managed by rule %q: %s", rule.Name, sourceAbs)
			}
		}
	}
	return nil
}

func applyAddSpec(doc *yaml.Node, configFile *config.ConfigFile, spec *addSpec) error {
	root := ensureDocumentMap(doc)
	rulesNode := ensureSequenceField(root, "rules")

	itemNode := newMapNode(
		newScalarNode("src"), newScalarNode(spec.ManagedSourceRel),
		newScalarNode("dest"), newScalarNode(spec.Dest),
	)

	if spec.AppendToRule {
		ruleNode := findRuleNode(rulesNode, spec.RuleName)
		if ruleNode == nil {
			return fmt.Errorf("failed to locate target rule %q in yaml", spec.RuleName)
		}
		itemsNode := ensureSequenceField(ruleNode, "items")
		itemsNode.Content = append(itemsNode.Content, itemNode)
		return nil
	}

	ruleNode := newMapNode(
		newScalarNode("name"), newScalarNode(spec.RuleName),
	)
	if shouldWriteRuleMode(configFile.Mode, spec.Mode) {
		ruleNode.Content = append(ruleNode.Content, newScalarNode("mode"), newScalarNode(string(spec.Mode)))
	}
	ruleNode.Content = append(ruleNode.Content, newScalarNode("items"), &yaml.Node{
		Kind:    yaml.SequenceNode,
		Style:   0,
		Content: []*yaml.Node{itemNode},
	})

	rulesNode.Content = append(rulesNode.Content, ruleNode)
	return nil
}

func shouldWriteRuleMode(globalMode, ruleMode config.Mode) bool {
	if globalMode == "" {
		return ruleMode != config.ModeSymlink
	}
	return globalMode != ruleMode
}

func writeYAMLDocument(path string, doc *yaml.Node) error {
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp config file: %w", err)
	}

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		enc.Close()
		f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to encode yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close yaml encoder: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp config file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to replace config file: %w", err)
	}
	return nil
}

func ensureDocumentMap(doc *yaml.Node) *yaml.Node {
	if doc.Kind == 0 {
		doc.Kind = yaml.DocumentNode
	}
	if len(doc.Content) == 0 || doc.Content[0] == nil {
		doc.Content = []*yaml.Node{newMapNode()}
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		doc.Content[0] = newMapNode()
	}
	return doc.Content[0]
}

func ensureSequenceField(mapNode *yaml.Node, key string) *yaml.Node {
	if node := mappingValue(mapNode, key); node != nil {
		if node.Kind != yaml.SequenceNode {
			node.Kind = yaml.SequenceNode
			node.Tag = "!!seq"
			node.Content = nil
		}
		return node
	}

	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	mapNode.Content = append(mapNode.Content, newScalarNode(key), seq)
	return seq
}

func mappingValue(mapNode *yaml.Node, key string) *yaml.Node {
	for idx := 0; idx+1 < len(mapNode.Content); idx += 2 {
		if mapNode.Content[idx].Value == key {
			return mapNode.Content[idx+1]
		}
	}
	return nil
}

func findRuleNode(rulesNode *yaml.Node, name string) *yaml.Node {
	for _, ruleNode := range rulesNode.Content {
		if ruleNode.Kind != yaml.MappingNode {
			continue
		}
		value := mappingValue(ruleNode, "name")
		if value != nil && strings.EqualFold(value.Value, name) {
			return ruleNode
		}
	}
	return nil
}

func newMapNode(content ...*yaml.Node) *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: content,
	}
}

func newScalarNode(value string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: value,
	}
}

func slugify(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

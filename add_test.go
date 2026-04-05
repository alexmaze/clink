package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexmaze/clink/config"
	"github.com/alexmaze/clink/lib/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseAddArgs(t *testing.T) {
	opts, err := parseAddArgs([]string{"-c", "./config.yaml", "--mode", "copy", "./.vimrc"})
	require.NoError(t, err)
	assert.Equal(t, "./config.yaml", opts.ConfigPath)
	assert.Equal(t, "copy", opts.Mode)
	assert.Equal(t, "./.vimrc", opts.Source)
}

func TestParseAddArgs_MissingConfig(t *testing.T) {
	_, err := parseAddArgs([]string{"./.vimrc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires -c")
}

func TestParseAddArgs_MissingSource(t *testing.T) {
	_, err := parseAddArgs([]string{"-c", "./config.yaml"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "usage")
}

func TestPlanManagedSourcePath_ImportsExternalFile(t *testing.T) {
	configDir := t.TempDir()
	sourceDir := t.TempDir()
	sourceAbs := filepath.Join(sourceDir, ".vimrc")
	require.NoError(t, os.WriteFile(sourceAbs, []byte("set nu"), 0644))

	rel, abs, err := planManagedSourcePath(configDir, sourceAbs, "vim config", fileutil.PathTypeFile)
	require.NoError(t, err)
	assert.Equal(t, "./.src/vim-config/.vimrc", rel)
	assert.Equal(t, filepath.Join(configDir, ".src", "vim-config", ".vimrc"), abs)
}

func TestPlanManagedSourcePath_KeepsSourceInsideConfigDir(t *testing.T) {
	configDir := t.TempDir()
	sourceAbs := filepath.Join(configDir, "dotfiles", ".vimrc")
	require.NoError(t, os.MkdirAll(filepath.Dir(sourceAbs), 0755))
	require.NoError(t, os.WriteFile(sourceAbs, []byte("set nu"), 0644))

	rel, abs, err := planManagedSourcePath(configDir, sourceAbs, "vim config", fileutil.PathTypeFile)
	require.NoError(t, err)
	assert.Equal(t, "./dotfiles/.vimrc", rel)
	assert.Equal(t, sourceAbs, abs)
}

func TestApplyAddSpec_AppendsNewRule(t *testing.T) {
	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			newMapNode(
				newScalarNode("mode"), newScalarNode("symlink"),
				newScalarNode("rules"), &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"},
			),
		},
	}
	configFile := &config.ConfigFile{Mode: config.ModeSymlink}

	err := applyAddSpec(doc, configFile, &addSpec{
		ConfigPath:       "/tmp/config.yaml",
		ConfigDir:        "/tmp",
		SourceAbs:        "/tmp/.vimrc",
		ManagedSourceAbs: "/tmp/.src/vim/.vimrc",
		ManagedSourceRel: "./.src/vim/.vimrc",
		Dest:             "/home/test/.vimrc",
		RuleName:         "vim",
		Mode:             config.ModeSymlink,
	})
	require.NoError(t, err)

	rulesNode := mappingValue(doc.Content[0], "rules")
	require.NotNil(t, rulesNode)
	require.Len(t, rulesNode.Content, 1)

	ruleNode := rulesNode.Content[0]
	assert.Equal(t, "vim", mappingValue(ruleNode, "name").Value)
	assert.Nil(t, mappingValue(ruleNode, "mode"))

	itemsNode := mappingValue(ruleNode, "items")
	require.NotNil(t, itemsNode)
	require.Len(t, itemsNode.Content, 1)
	assert.Equal(t, "./.src/vim/.vimrc", mappingValue(itemsNode.Content[0], "src").Value)
	assert.Equal(t, "/home/test/.vimrc", mappingValue(itemsNode.Content[0], "dest").Value)
}

func TestApplyAddSpec_AppendsItemToExistingRule(t *testing.T) {
	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			newMapNode(
				newScalarNode("rules"), &yaml.Node{
					Kind: yaml.SequenceNode,
					Tag:  "!!seq",
					Content: []*yaml.Node{
						newMapNode(
							newScalarNode("name"), newScalarNode("vim"),
							newScalarNode("items"), &yaml.Node{
								Kind: yaml.SequenceNode,
								Tag:  "!!seq",
								Content: []*yaml.Node{
									newMapNode(
										newScalarNode("src"), newScalarNode("./.src/vim/.vimrc"),
										newScalarNode("dest"), newScalarNode("/home/test/.vimrc"),
									),
								},
							},
						),
					},
				},
			),
		},
	}
	configFile := &config.ConfigFile{
		Rules: []*config.Rule{{
			Name: "vim",
			Mode: config.ModeSymlink,
		}},
	}

	err := applyAddSpec(doc, configFile, &addSpec{
		ConfigPath:       "/tmp/config.yaml",
		ConfigDir:        "/tmp",
		SourceAbs:        "/tmp/.zshrc",
		ManagedSourceAbs: "/tmp/.src/vim/.zshrc",
		ManagedSourceRel: "./.src/vim/.zshrc",
		Dest:             "/home/test/.zshrc",
		RuleName:         "vim",
		Mode:             config.ModeSymlink,
		AppendToRule:     true,
	})
	require.NoError(t, err)

	rulesNode := mappingValue(doc.Content[0], "rules")
	require.NotNil(t, rulesNode)
	itemsNode := mappingValue(rulesNode.Content[0], "items")
	require.Len(t, itemsNode.Content, 2)
	assert.Equal(t, "/home/test/.zshrc", mappingValue(itemsNode.Content[1], "dest").Value)
}

func TestRunAdd_ImportsFileAndUpdatesConfig(t *testing.T) {
	workDir := t.TempDir()
	configPath := filepath.Join(workDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("rules: []\n"), 0644))

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, ".vimrc")
	require.NoError(t, os.WriteFile(sourcePath, []byte("set nu"), 0644))

	err := runAdd(AddOpts{
		ConfigPath: configPath,
		Source:     sourcePath,
		Yes:        true,
	})
	require.NoError(t, err)

	managedPath := filepath.Join(workDir, ".src", "vimrc", ".vimrc")
	gotManaged, err := os.ReadFile(managedPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("set nu"), gotManaged)

	configFile, err := config.ParseConfigFileOnly(configPath)
	require.NoError(t, err)
	require.Len(t, configFile.Rules, 1)
	assert.Equal(t, ".vimrc", configFile.Rules[0].Name)
	assert.Equal(t, config.ModeSymlink, configFile.Rules[0].Mode)
	assert.Equal(t, managedPath, configFile.Rules[0].Items[0].Source)
	assert.Equal(t, sourcePath, configFile.Rules[0].Items[0].Destination)
}

func TestRunAdd_AppendsToExistingRule(t *testing.T) {
	workDir := t.TempDir()
	configPath := filepath.Join(workDir, "config.yaml")
	configBody := `mode: copy
rules:
  - name: shell
    items:
      - src: ./.src/shell/.bashrc
        dest: /tmp/.bashrc
`
	require.NoError(t, os.MkdirAll(filepath.Join(workDir, ".src", "shell"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, ".src", "shell", ".bashrc"), []byte("alias ll"), 0644))
	require.NoError(t, os.WriteFile(configPath, []byte(configBody), 0644))

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, ".zshrc")
	require.NoError(t, os.WriteFile(sourcePath, []byte("export EDITOR=vim"), 0644))

	err := runAdd(AddOpts{
		ConfigPath: configPath,
		Source:     sourcePath,
		Rule:       "shell",
		Dest:       "/tmp/.zshrc",
		Yes:        true,
	})
	require.NoError(t, err)

	configFile, err := config.ParseConfigFileOnly(configPath)
	require.NoError(t, err)
	require.Len(t, configFile.Rules, 1)
	require.Len(t, configFile.Rules[0].Items, 2)
	assert.Equal(t, config.ModeCopy, configFile.Rules[0].Mode)
	assert.Equal(t, "/tmp/.zshrc", configFile.Rules[0].Items[1].Destination)
	assert.Equal(t, filepath.Join(workDir, ".src", "shell", ".zshrc"), configFile.Rules[0].Items[1].Source)
}

func TestRunAdd_RejectsDuplicateDest(t *testing.T) {
	workDir := t.TempDir()
	configPath := filepath.Join(workDir, "config.yaml")
	configBody := `rules:
  - name: vim
    items:
      - src: ./.src/vim/.vimrc
        dest: /tmp/.vimrc
`
	require.NoError(t, os.MkdirAll(filepath.Join(workDir, ".src", "vim"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, ".src", "vim", ".vimrc"), []byte("set nu"), 0644))
	require.NoError(t, os.WriteFile(configPath, []byte(configBody), 0644))

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, ".gvimrc")
	require.NoError(t, os.WriteFile(sourcePath, []byte("set mouse=a"), 0644))

	err := runAdd(AddOpts{
		ConfigPath: configPath,
		Source:     sourcePath,
		Dest:       "/tmp/.vimrc",
		Yes:        true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destination already exists")
}

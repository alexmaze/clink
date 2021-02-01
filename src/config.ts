import * as fs from "fs"
import * as path from "path"
import * as prompts from "prompts"
import { parse } from "yaml"
import { logTitle } from "display"

export interface IConfig {
  workDir: string // 配置文件所在目录
  dryRun: boolean
  backupPath: string

  vars: IVars
  rules: IRule[]
}

// NewConfig 初始化配置
export async function NewConfig(cmdProps: ICmdProps): Promise<IConfig> {
  const configFilePath = absPath(process.cwd(), cmdProps.config)
  const dryRun = cmdProps.dryRun || false
  const workDir = path.dirname(configFilePath)
  let vars: IVars = {}
  let rules: IRule[] = []

  // 读取配置文件
  let configStr: string
  try {
    const buf = fs.readFileSync(configFilePath)
    configStr = buf.toString()
  } catch (err) {
    console.error("failed to read config file: " + configFilePath, err)
    throw err
  }

  // 解析配置文件
  let rawConfig: IConfigFile
  try {
    rawConfig = parse(configStr)
  } catch (err) {
    console.error("failed to parse config file as yaml: " + configFilePath, err)
    throw err
  }

  // 1. 确认所有变量
  logTitle("# Please confirm variables (type to modify):")
  if (rawConfig.vars) {
    for (; true; ) {
      const resp = await prompts(
        Object.keys(rawConfig.vars).map(key => ({
          name: key,
          type: "text",
          message: `${key}`,
          initial: rawConfig.vars[key]
        }))
      )

      const { ok } = await prompts({
        type: "confirm",
        name: "ok",
        message: "Is that ok?",
        initial: false
      })

      if (ok) {
        vars = resp
        break
      }
    }
  }

  // 2. 转换 Rules: 变量替换 & 绝对路径 & src 检查
  for (const r of rawConfig.rules) {
    const items: IRuleItem[] = []
    for (const ri of r.items) {
      items.push(await newRuleItem(vars, workDir, ri))
    }

    const rule: IRule = {
      name: r.name,
      items: items
    }
    rules.push(rule)
  }

  // 3. 备份目录
  let backupPath: string
  for (; true; ) {
    const res = await prompts({
      type: "text",
      name: "backupPath",
      initial: path.join(workDir, "backup", `${new Date().toISOString()}`),
      message: "Backup location"
    })

    backupPath = absPath(process.cwd(), res.backupPath)
    try {
      fs.mkdirSync(backupPath, { recursive: true })
      break
    } catch (err) {
      console.error("failed to create backup folder", err)
    }
  }

  return {
    workDir,
    dryRun,
    backupPath,
    vars,
    rules
  }
}

async function newRuleItem(
  vars: IVars,
  workDir: string,
  r: {
    src: string
    dest: string
  }
): Promise<IRuleItem> {
  let src = absPath(workDir, replaceVarsInStr(vars, r.src))
  let dest = absPath(workDir, replaceVarsInStr(vars, r.dest))
  let mode: "file" | "dir"

  try {
    const srcInfo = fs.statSync(src)

    if (srcInfo.isFile()) {
      mode = "file"
    } else if (srcInfo.isDirectory()) {
      mode = "dir"
    } else {
      throw new Error("invalid src type: , only support file or folder" + r.src)
    }
  } catch (err) {
    console.error("failed to locate src: " + r.src, err)
    throw err
  }

  return {
    src,
    dest,
    mode
  }
}
// 命令行参数
export interface ICmdProps {
  dryRun?: boolean
  config: string
}

// 配置文件内容
interface IConfigFile {
  vars: IVars
  rules: {
    name: string
    items: {
      src: string
      dest: string
    }[]
  }[]
}

export interface IVars {
  [var_name: string]: string
}

export interface IRule {
  name: string
  items: IRuleItem[]
}

export interface IRuleItem {
  mode: "file" | "dir"
  src: string
  dest: string
}

function absPath(ctxPath: string, p: string): string {
  // 绝对路径
  path.resolve(p)
  if (p.startsWith("/")) {
    return path.normalize(p)
  }

  // home 相对路径
  if (p.startsWith("~")) {
    if (!process.env.HOME) {
      throw new Error("can not determine `~` in path")
    }
    return path.join(process.env.HOME, p.substr(1))
  }

  // ctxPath 相对路径
  return path.join(ctxPath, p)
}

function replaceVarsInStr(
  vars: { [var_name: string]: string },
  originStr: string
): string {
  for (const varKey in vars) {
    const regex = new RegExp("\\${" + varKey + "}", "g")
    originStr = originStr.replace(regex, vars[varKey])
  }

  originStr = originStr.replace(/\/+/g, "/")
  return originStr
}

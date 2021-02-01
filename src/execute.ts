import * as ora from "ora"
import * as path from "path"
import * as fs from "fs"
import { IConfig, IRule, IRuleItem } from "config"

export interface IRuleItemResult {
  item: IRuleItem
  err?: string
}

export interface IRuleResult {
  rule: IRule
  itemResults: IRuleItemResult[]
}

// Execute 根据配置文件执行
export async function Execute(cfg: IConfig): Promise<IRuleResult[]> {
  const results: IRuleResult[] = []

  const sp = ora(`execution start...`).start()

  for (const r of cfg.rules) {
    results.push(await doExecuteRule(cfg, sp, r))
  }

  sp.succeed(`${results.length} rules executed.`)

  return results
}

async function doExecuteRule(
  cfg: IConfig,
  sp: ora.Ora,
  rule: IRule
): Promise<IRuleResult> {
  sp.text = `[${rule.name}] start`

  const res: IRuleResult = {
    rule,
    itemResults: []
  }

  for (const item of rule.items) {
    res.itemResults.push(await doExecuteRuleItem(cfg, sp, rule, item))
  }

  sp.text = `[${rule.name}] finish`

  return res
}

async function doExecuteRuleItem(
  cfg: IConfig,
  sp: ora.Ora,
  rule: IRule,
  item: IRuleItem
): Promise<IRuleItemResult> {
  const res: IRuleItemResult = {
    item: item,
    err: undefined
  }

  try {
    // 1. 初始化目录
    const destPath = path.dirname(item.dest)
    sp.text = `[${rule.name}] mkdir ${destPath}`
    fs.mkdirSync(destPath, { recursive: true })

    // 2. 备份原文件
    const targetBackupPath = path.join(cfg.backupPath, item.dest)
    const targetBackupDir = path.dirname(targetBackupPath)

    if (fs.existsSync(item.dest)) {
      sp.text = `[${rule.name}] backup ${item.dest}`
      fs.mkdirSync(targetBackupDir, { recursive: true })
      try {
        fs.renameSync(item.dest, targetBackupPath)
      } catch (err) {
        console.error("failed to backup original config " + item.dest)
        res.err = err
        return res
      }
    }

    // 3. 创建软连接
    sp.text = `[${rule.name}] link ${item.src} -> ${item.dest}`
    fs.symlinkSync(item.src, item.dest)
  } catch (err) {
    console.error(err)
    res.err = err
  }

  return res
}

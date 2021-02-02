import * as chalk from "chalk"
import { IConfig } from "./config"
import { IRuleResult } from "./execute"

export function DisplayResults(cfg: IConfig, res: IRuleResult[]) {
  logTitle(`\n# ${cfg.dryRun ? "dry run" : "execution"} results:`)
  for (const r of res) {
    logTitle2(`[${r.rule.name}]`)
    for (const i of r.itemResults) {
      if (i.err) {
        logTitle3_failed(`   ${i.item.src}`)
        logTitle3_failed(` ↳ ${i.item.dest}`)
      } else {
        logTitle3_success(`   ${i.item.src}`)
        logTitle3_success(` ↳ ${i.item.dest}`)
      }
    }
  }
}

export function logTitle(str: string) {
  console.log(chalk.blue(chalk.bold(str)))
}

export function logTitle2(str: string) {
  console.log(chalk.cyan(str))
}

export function logTitle3_success(str: string) {
  console.log(chalk.green(str))
}

export function logTitle3_failed(str: string) {
  console.log(chalk.red(str))
}

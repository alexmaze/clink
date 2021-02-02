import { Command } from "commander"
import { Execute } from "./execute"
import { DisplayResults } from "./display"
import { ICmdProps, NewConfig } from "./config"

const cmd = new Command()

cmd
  .version("0.0.1")
  .description("A configuration file centralized management tool.")
  .option(
    "-d, --dry-run",
    "dry-run mode, will display all changes will be made"
  )
  .requiredOption(
    "-c, --config <CONFIG_FILE>",
    "specify config file path, e.g. `./config.yaml`"
  )
  .action(async (props: ICmdProps) => {
    const cfg = await NewConfig(props)
    const results = await Execute(cfg)
    DisplayResults(cfg, results)
  })

cmd.parse(process.argv)

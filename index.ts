#!/usr/bin/env deno run -A --unstable

import { NewConfig } from "./config.ts";
import { Execute } from "./execute.ts";
import { getBackupDir } from "./util.ts";

try {
  const c = await NewConfig(Deno.args[0]);

  const backupPath = await getBackupDir();

  for (const r of c.rules) {
    await Execute(backupPath, r);
  }

  console.log("DONE.");
  console.log("原始文件备份路径", backupPath);
} catch (err) {
  console.error(err);
  Deno.exit(1);
}

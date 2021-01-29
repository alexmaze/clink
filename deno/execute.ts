import { IRule, IRuleItem } from "./config.ts";
import { dirname, join } from "https://deno.land/std/path/mod.ts";
import { exists } from "https://deno.land/std/fs/mod.ts";

export async function Execute(backupPath: string, rule: IRule): Promise<void> {
  console.log("start", rule.name);
  if (rule.files) {
    for (const ri of rule.files) {
      await doExec(backupPath, ri, "file");
    }
  }

  if (rule.folders) {
    for (const ri of rule.folders) {
      await doExec(backupPath, ri, "dir");
    }
  }
  return;
}

async function doExec(
  backupPath: string,
  item: IRuleItem,
  type: "file" | "dir"
) {
  // 1. 初始化目录
  await Deno.mkdir(dirname(item.dist), { recursive: true });

  // 2. 备份原文件
  if (await exists(item.dist)) {
    await Deno.mkdir(dirname(join(backupPath, item.dist)), { recursive: true });
    await Deno.rename(item.dist, join(backupPath, item.dist));
  }

  // 3. 创建软连接
  await Deno.symlink(item.src, item.dist, { type });
}

import { parse } from "https://deno.land/std/encoding/yaml.ts";
import {
  join,
  resolve,
  dirname,
  isAbsolute,
} from "https://deno.land/std/path/mod.ts";

export interface IConfig {
  work_dir: string; // config 文件目录，运行时自动生成
  rules: IRule[];
}

export interface IRule {
  name: string;
  files: IRuleItem[];
  folders: IRuleItem[];
}

export interface IRuleItem {
  src: string;
  dist: string;
}

export async function NewConfig(path: string): Promise<IConfig> {
  let c: IConfig;

  if (!path) {
    throw "请指定配置文件";
  }

  path = resolve(path);

  // 读取文件
  try {
    const text = await Deno.readTextFile(path);
    c = parse(text) as IConfig;
  } catch (err) {
    console.log(err);
    throw "读取文件失败";
  }

  c.work_dir = path;

  // 处理配置内容：
  // 1. 将路径转换为绝对路径
  // 2. 检查所有 src 文件（夹）是否存在
  for (const r of c.rules) {
    if (r.files) {
      for (const ri of r.files) {
        await standardRuleItem("file", dirname(path), ri);
      }
    }
    if (r.folders) {
      for (const ri of r.folders) {
        await standardRuleItem("folder", dirname(path), ri);
      }
    }
  }

  return c;
}

async function standardRuleItem(
  type: "file" | "folder",
  relativePath: string,
  ruleItem: IRuleItem
) {
  // 1. 转换路径
  ruleItem.src = absPath(relativePath, ruleItem.src);
  ruleItem.dist = absPath(relativePath, ruleItem.dist);

  // 2. 检查文件是否存在
  let st: Deno.FileInfo;
  try {
    st = await Deno.statSync(ruleItem.src);
  } catch (err) {
    throw "找不到文件 " + ruleItem.src;
  }

  if (type === "file" && !st.isFile) {
    throw "不是文件" + ruleItem.src;
  }

  if (type === "folder" && !st.isDirectory) {
    throw "不是文件夹" + ruleItem.src;
  }

  return;
}

function absPath(relativePath: string, path: string): string {
  if (isAbsolute(path)) {
    return path;
  }

  if (path.charCodeAt(0) === 126) {
    const homePath = Deno.env.get("HOME") as string;
    return join(homePath, path.substring(1));
  }

  return join(relativePath, path);
}

import { parse } from "https://deno.land/std/encoding/yaml.ts";
import {
  join,
  resolve,
  dirname,
  isAbsolute,
} from "https://deno.land/std/path/mod.ts";
import Ask from "https://deno.land/x/ask/mod.ts";

export interface IVars {
  [var_name: string]: string; // 值为默认值
}

export interface IConfig {
  work_dir: string; // config 文件目录，运行时自动生成
  vars: IVars;
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

  let configText;
  // 读取文件
  try {
    configText = await Deno.readTextFile(path);
    c = parse(configText) as IConfig;
  } catch (err) {
    console.log(err);
    throw "读取文件失败";
  }

  c.work_dir = path;

  // 处理配置内容：
  // 1. 确认所有变量值，并进行变量替换
  // 2. 将路径转换为绝对路径
  // 3. 检查所有 src 文件（夹）是否存在
  if (c.vars) {
    for (; true; ) {
      console.log("\n请确认或修改以下变量值，回车保持默认：");
      const ask = new Ask();
      const answers = await ask.prompt(
        Object.keys(c.vars).map((key) => ({
          name: key,
          type: "input",
          message: `${key} = [${c.vars[key]}]`,
        }))
      );

      console.log("\n------------------------------------");
      for (const key in answers) {
        if (answers[key]) {
          c.vars[key] = answers[key];
        }
        console.log(`${key} = ${c.vars[key]}`);
      }
      console.log("------------------------------------\n");

      const { confirmed } = await ask.prompt([
        {
          name: "confirmed",
          type: "input",
          message: "Y/y 确认，其他任意键返回重新填写",
        },
      ]);

      if (confirmed === "y" || confirmed === "Y") {
        break;
      }
    }
  } else {
    c.vars = {}
  }

  for (const r of c.rules) {
    if (r.files) {
      for (const ri of r.files) {
        await standardRuleItem(c.vars, "file", dirname(path), ri);
      }
    }

    if (r.folders) {
      for (const ri of r.folders) {
        await standardRuleItem(c.vars, "folder", dirname(path), ri);
      }
    }
  }

  return c;
}

function fillVars(vars: IVars, originStr: string): string {

  // 0. 变量替换
  for (const varKey in vars) {
    originStr = originStr.replaceAll(`\${${varKey}}`, vars[varKey])
  }

  originStr = originStr.replace(/\/+/g, "/")
  console.log("---", originStr)
  return originStr
}

async function standardRuleItem(
  vars: IVars,
  type: "file" | "folder",
  relativePath: string,
  ruleItem: IRuleItem
) {

  // 1. 转换路径
  ruleItem.src = absPath(relativePath, fillVars(vars, ruleItem.src));
  ruleItem.dist = absPath(relativePath, fillVars(vars, ruleItem.dist));

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

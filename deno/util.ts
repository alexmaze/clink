import {
  join,
  resolve,
  dirname,
  isAbsolute,
} from "https://deno.land/std/path/mod.ts";

export function sleep(sec: number) {
  return new Promise((res) => {
    setTimeout(res, sec * 1000);
  });
}

// TODO 目前是 <user_home>/.config-manager/backup_<timestamp>/
export async function getBackupDir() {
  const homePath = Deno.env.get("HOME") as string;

  const path = join(
    homePath,
    ".config-manager",
    `backup_${new Date().toISOString()}`
  );

  await Deno.mkdir(path, { recursive: true });

  return path;
}

import commonjs from "@rollup/plugin-commonjs"
import typescript from "@rollup/plugin-typescript"
import { nodeResolve } from "@rollup/plugin-node-resolve"

export default {
  input: ["src/index.ts"],
  output: {
    banner: "#!/usr/bin/env node",
    file: "bin/clink.js",
    format: "cjs"
  },
  plugins: [typescript(), nodeResolve(), commonjs()]
}

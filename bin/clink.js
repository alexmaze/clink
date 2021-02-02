#!/usr/bin/env node
'use strict';

Object.defineProperty(exports, "__esModule", { value: true });
var tslib_1 = require("tslib");
var commander_1 = require("commander");
var execute_1 = require("./execute");
var display_1 = require("./display");
var config_1 = require("./config");
var cmd = new commander_1.Command();
cmd
    .version("0.0.1")
    .description("A configuration file centralized management tool.")
    .option("-d, --dry-run", "dry-run mode, will display all changes will be made")
    .requiredOption("-c, --config <CONFIG_FILE>", "specify config file path, e.g. `./config.yaml`")
    .action(function (props) { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
    var cfg, results;
    return tslib_1.__generator(this, function (_a) {
        switch (_a.label) {
            case 0: return [4 /*yield*/, config_1.NewConfig(props)];
            case 1:
                cfg = _a.sent();
                return [4 /*yield*/, execute_1.Execute(cfg)];
            case 2:
                results = _a.sent();
                display_1.DisplayResults(cfg, results);
                return [2 /*return*/];
        }
    });
}); });
cmd.parse(process.argv);

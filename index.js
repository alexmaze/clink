const commander = require("commander");

commander
  .version("0.0.1")
  .description("A cli application named pro")
  .option("-p, --peppers", "Add peppers")
  .option("-P, --pineapple", "Add pineapple")
  .option("-b, --bbq-sauce", "Add bbq sauce")
  .option(
    "-c, --cheese [type]",
    "Add the specified type of cheese [marble]",
    "marble"
  );

commander.parse(process.argv);

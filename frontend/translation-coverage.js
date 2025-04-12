const fs = require("fs");
const path = require("path");

// Helper to flatten nested keys
function flatten(obj, prefix = "") {
  return Object.keys(obj).reduce((acc, k) => {
    const pre = prefix.length ? `${prefix}.` : "";
    if (typeof obj[k] === "object" && obj[k] !== null) {
      Object.assign(acc, flatten(obj[k], pre + k));
    } else {
      acc[pre + k] = obj[k];
    }
    return acc;
  }, {});
}

const baseLocale = "en";
const localesDir = path.join(__dirname, "messages");

const baseMessages = flatten(
  JSON.parse(fs.readFileSync(`${localesDir}/${baseLocale}.json`, "utf-8"))
);
const baseKeys = Object.keys(baseMessages);

fs.readdirSync(localesDir).forEach((file) => {
  const locale = path.basename(file, ".json");
  if (locale === baseLocale) return;

  const targetMessages = flatten(
    JSON.parse(fs.readFileSync(`${localesDir}/${file}`, "utf-8"))
  );
  const translatedKeys = Object.keys(targetMessages);

  const translatedCount = baseKeys.filter((key) => targetMessages[key]).length;
  const total = baseKeys.length;
  const percentage = ((translatedCount / total) * 100).toFixed(2);

  console.log(
    `${locale}: ${percentage}% translated (${translatedCount}/${total})`
  );
});

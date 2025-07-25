const fs = require("fs");
const path = require("path");

if (process.stdout.isTTY) {
  color_reset = "\x1b[0m";
  color_yellow = "\x1b[33m";
  color_green = "\x1b[32m";
  color_red = "\x1b[31m";
  color_cyan = "\x1b[36m";
} else {
  color_reset = "";
  color_yellow = "";
  color_green = "";
  color_red = "";
  color_cyan = "";
}

if (process.argv.includes("--help") || process.argv.includes("-h")) {
  console.log(`-h, --help                 Print this help message`);
  console.log(`-u, --update-translations  Update translation files in the frontend/messages directory with missing keys`);
  process.exit(0);
}

const argUpdateTranslations = process.argv.includes("--update-translations") || process.argv.includes("-u");

console.log(`${color_cyan}Translation coverage report for frontend/messages/*.json${color_reset}`);

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

function deepMerge(target, source) {
    if (typeof target !== 'object' || target === null || typeof source !== 'object' || source === null) {
        throw new TypeError('Both arguments must be objects');
    }

    for (const key of Object.keys(source)) {
        const srcValue = source[key];
        const tgtValue = target[key];

        if (tgtValue && typeof tgtValue === 'object' && !Array.isArray(tgtValue) &&
            srcValue && typeof srcValue === 'object' && !Array.isArray(srcValue)) {

            // Recursively merge nested objects
            deepMerge(tgtValue, srcValue);
        } else {
            // Set the property if it doesn't exist in target or if it's not an object
            target[key] = srcValue;
        }
    }

    return target;
}

const baseLocale = "en";
const localesDir = path.join(__dirname, "messages");

const baseLanguageObject = JSON.parse(fs.readFileSync(`${localesDir}/${baseLocale}.json`, "utf-8"));
const baseMessages = flatten(structuredClone(baseLanguageObject));
const baseKeys = Object.keys(baseMessages);

let translationStats = {};
let extraKeysInTargetLocales = {};
let missingKeysInTargetLocales = {};
let updatedTargetTranslationFiles = [];
fs.readdirSync(localesDir).forEach((file) => {
  const locale = path.basename(file, ".json");
  if (locale === baseLocale) return;

  const targetLanguageFile = `${localesDir}/${file}`;
  const targetLanguageObject = JSON.parse(fs.readFileSync(targetLanguageFile, "utf-8"));
  if (argUpdateTranslations) {
    let newTargetLanguageObject = structuredClone(baseLanguageObject)
    deepMerge(newTargetLanguageObject, targetLanguageObject);
    fs.writeFileSync(
      targetLanguageFile,
      JSON.stringify(newTargetLanguageObject, null, 2),
      { encoding: "utf-8" },
    );
    updatedTargetTranslationFiles.push(targetLanguageFile);
  }
  const targetMessages = flatten(structuredClone(targetLanguageObject));
  extraKeysInTargetLocales[locale] = Object.keys(targetMessages).filter(
    (key) => !baseKeys.includes(key)
  );
  const translatedKeys = Object.keys(targetMessages);
  const missingKeys = baseKeys.filter((key) => !translatedKeys.includes(key));
  if (missingKeys.length > 0) {
    missingKeysInTargetLocales[locale] = missingKeys;
  }

  const translatedCount = baseKeys.filter((key) => targetMessages[key]).length;
  const baseLanguageKeyCount = baseKeys.length;
  const percentageTranslated = ((translatedCount / baseLanguageKeyCount) * 100).toFixed(2);

  translationStats[locale] = {
    baseLanguageKeyCount,
    missingKeys,
    percentageTranslated,
    translatedCount,
  };
});
if (Object.keys(missingKeysInTargetLocales).length > 0) {
  console.log("Missing translations:");
  Object.entries(missingKeysInTargetLocales).forEach(([locale, keys]) => {
    console.log(`${locale}:`);
    keys.forEach((key) => {
      console.log(`  - ${key}`);
    });
  });
  console.log(); // add final newline for better readability
} else {
  console.log("🎉 All translations are present in all locales 🎉\n");
}

const localesWithExtraKeys = Object.keys(extraKeysInTargetLocales).filter(
  (locale) => extraKeysInTargetLocales[locale].length > 0
);
if (localesWithExtraKeys.length > 0) {
  console.warn("Warning: The following locales have extra translations that are missing from the base locale");
  localesWithExtraKeys.forEach((locale) => {
    console.warn(`${locale}:`);
    extraKeysInTargetLocales[locale].forEach((key) => {
      console.warn(`  - ${key}`);
    });
  });
  // add final newline for better readability
  console.log('');
}

if (argUpdateTranslations) {
  console.log(`${color_yellow}${updatedTargetTranslationFiles.length}${color_reset} translation files have been updated, please translate them:`);
  updatedTargetTranslationFiles.forEach((file) => console.log(`- ${file}`));
  // add final newline for better readability
  console.log('');
}

console.log("Translation stats:");
Object.keys(translationStats).forEach((locale) => {
  const { baseLanguageKeyCount, missingKeys, percentageTranslated, translatedCount } = translationStats[locale];

  let percentage_text = '';
  if (percentageTranslated == 100 && locale !== "en") {
    percentage_text = `${color_green}100${color_reset}`;
  }
  if (percentageTranslated < 100 && locale !== "en") {
    percentage_text = `${color_yellow}${percentageTranslated}${color_reset}`;
  }
  if (percentageTranslated < 50 && locale !== "en") {
    percentage_text = `${color_red}${percentageTranslated}${color_reset}`;
  }

  let missingKeysText = '';
  if (missingKeys.length > 0) {
    missingKeysText = `${color_yellow}${missingKeys.length}${color_reset}`;
  } else {
    missingKeysText = `${color_green}0${color_reset}`;
  }

  let extraKeysText = '';
  if (extraKeysInTargetLocales[locale].length > 0) {
    extraKeysText = `${color_red}${extraKeysInTargetLocales[locale].length}${color_reset}`;
  } else {
    extraKeysText = `${color_green}0${color_reset}`;
  }

  console.log(`${locale}: ${percentage_text}% translated (${translatedCount}/${baseLanguageKeyCount}) with ${missingKeysText} missing translations and ${extraKeysText} extra translations`);
});

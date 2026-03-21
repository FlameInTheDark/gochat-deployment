import { appendFileSync } from "node:fs";
import semanticRelease from "semantic-release";

const outputs = {
  released: "false",
  version: "",
  git_tag: ""
};

try {
  const result = await semanticRelease();

  if (result?.nextRelease) {
    outputs.released = "true";
    outputs.version = result.nextRelease.version ?? "";
    outputs.git_tag = result.nextRelease.gitTag ?? "";
  }
} catch (error) {
  console.error(error);
  process.exit(1);
}

if (process.env.GITHUB_OUTPUT) {
  appendFileSync(
    process.env.GITHUB_OUTPUT,
    `released=${outputs.released}\nversion=${outputs.version}\ngit_tag=${outputs.git_tag}\n`
  );
}

if (outputs.released === "true") {
  console.log(`Published ${outputs.git_tag}`);
} else {
  console.log("No release published");
}

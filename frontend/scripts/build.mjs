import { cp, mkdir, rm } from "node:fs/promises";

await rm("dist", { recursive: true, force: true });
await mkdir("dist/assets", { recursive: true });
await cp("public/index.html", "dist/index.html");
await cp("public/styles.css", "dist/styles.css");
await cp("public/test_data", "dist/test_data", { recursive: true });

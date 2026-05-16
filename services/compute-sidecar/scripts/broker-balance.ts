import { loadEnvFile } from "../src/load-env.js";

loadEnvFile();

const stubMode = (process.env.STUB_MODE ?? "true") === "true";

if (stubMode) {
  console.log(JSON.stringify({ status: "stub", balance: "0" }));
} else {
  console.log(
    JSON.stringify({
      status: "configure",
      message:
        "Use the 0G broker funding workflow with local OG_RPC_URL and ZG_SERVING_BROKER_PRIVATE_KEY values.",
    }),
  );
}

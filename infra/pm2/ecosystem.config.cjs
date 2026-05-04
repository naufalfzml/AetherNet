module.exports = {
  apps: [
    {
      name: "aethernet-backend",
      cwd: "backend",
      script: "go",
      args: "run ./cmd/server",
      env: {
        STUB_MODE: "true",
      },
    },
    {
      name: "aethernet-indexer",
      cwd: "backend",
      script: "go",
      args: "run ./cmd/indexer",
      env: {
        STUB_MODE: "true",
      },
    },
  ],
};

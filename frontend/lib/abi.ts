export const agentINFTAbi = [
  {
    type: "function",
    name: "mintAgent",
    stateMutability: "payable",
    inputs: [
      { name: "metadataPointer", type: "string" },
      { name: "promptHash", type: "bytes32" },
    ],
    outputs: [
      { name: "tokenId", type: "uint256" },
      { name: "treasury", type: "address" },
    ],
  },
] as const;

export const treasuryAbi = [
  {
    type: "function",
    name: "buyShares",
    stateMutability: "payable",
    inputs: [
      { name: "amount", type: "uint256" },
      { name: "maxPrice", type: "uint256" },
    ],
    outputs: [],
  },
  {
    type: "function",
    name: "sellShares",
    stateMutability: "nonpayable",
    inputs: [
      { name: "amount", type: "uint256" },
      { name: "minPrice", type: "uint256" },
    ],
    outputs: [],
  },
  {
    type: "function",
    name: "claimDividends",
    stateMutability: "nonpayable",
    inputs: [],
    outputs: [],
  },
] as const;

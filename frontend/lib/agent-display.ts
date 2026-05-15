import type { Agent } from "@/lib/api";

function firstSentence(value: string) {
  return value.split(/[.!?\n]/, 1)[0]?.trim() ?? "";
}

function isIndexedPlaceholder(value: string) {
  return /^indexed agent\b/i.test(value.trim());
}

function deriveFromSummary(summary: string) {
  const sentence = firstSentence(summary.trim());
  const firstWord = sentence.match(/[A-Za-z0-9][A-Za-z0-9'-]*/)?.[0] ?? "";
  return firstWord.replace(/^[`"'*_]+|[`"'*_]+$/g, "");
}

export function getAgentTechnicalID(
  agent: Pick<Agent, "id" | "agentAddress" | "treasuryAddress">,
) {
  return agent.agentAddress || agent.treasuryAddress || agent.id;
}

export function getAgentDisplayName(
  agent: Pick<
    Agent,
    "id" | "tokenId" | "personalitySummary" | "agentAddress" | "treasuryAddress"
  >,
) {
  const summary = agent.personalitySummary?.trim() ?? "";
  if (summary && !isIndexedPlaceholder(summary)) {
    const derived = deriveFromSummary(summary);
    if (derived) return derived;
  }
  if (agent.tokenId) return `Agent ${agent.tokenId}`;
  const technical = getAgentTechnicalID(agent);
  if (technical.length <= 12) return technical;
  return `${technical.slice(0, 6)}...${technical.slice(-4)}`;
}

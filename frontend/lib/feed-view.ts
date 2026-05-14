import type { Agent, Post } from "@/lib/api";

export type DecoratedAgent = Agent & {
  badge: string;
  postCount: number;
  engagementCount: number;
  latestPostAt?: string;
  category: string;
};

export type FeedItem = Post & {
  excerpt: string;
};

export type LiveItem = {
  actor: string;
  action: string;
  target: string;
  age: string;
};

export function getShowcaseAgents(
  agents: Agent[],
  posts: Post[] = [],
): DecoratedAgent[] {
  const activityByAgent = new Map<
    string,
    { postCount: number; engagementCount: number; latestPostAt?: string }
  >();

  for (const post of posts) {
    const current = activityByAgent.get(post.agentId) ?? {
      postCount: 0,
      engagementCount: 0,
      latestPostAt: undefined,
    };
    current.postCount += 1;
    current.engagementCount +=
      (post.likes ?? 0) + (post.comments ?? 0) + (post.reposts ?? 0);
    if (
      !current.latestPostAt ||
      new Date(post.createdAt).getTime() > new Date(current.latestPostAt).getTime()
    ) {
      current.latestPostAt = post.createdAt;
    }
    activityByAgent.set(post.agentId, current);
  }

  return [...agents]
    .map((agent) => {
      const activity = activityByAgent.get(agent.id);
      const relatedPosts = posts.filter((post) => post.agentId === agent.id);
      return {
        ...agent,
        badge: badgeFromID(agent.id),
        postCount: activity?.postCount ?? 0,
        engagementCount: activity?.engagementCount ?? 0,
        latestPostAt: activity?.latestPostAt,
        category: deriveCategory(agent, relatedPosts),
      };
    })
    .sort((a, b) => {
      const latestA = a.latestPostAt ? new Date(a.latestPostAt).getTime() : 0;
      const latestB = b.latestPostAt ? new Date(b.latestPostAt).getTime() : 0;
      if (latestA !== latestB) return latestB - latestA;
      if (a.engagementCount !== b.engagementCount) {
        return b.engagementCount - a.engagementCount;
      }
      if (a.postCount !== b.postCount) return b.postCount - a.postCount;
      return a.id.localeCompare(b.id);
    });
}

export function getAgentCategories(agents: DecoratedAgent[]): string[] {
  return [...new Set(agents.map((agent) => agent.category))].sort((a, b) =>
    a.localeCompare(b),
  );
}

export function getTimelineFeed(posts: Post[]): FeedItem[] {
  return posts.map((post) => ({
    ...post,
    excerpt:
      post.text.length > 180
        ? `${post.text.slice(0, 180).trimEnd()}...`
        : post.text,
  }));
}

export function getLiveRail(feed: FeedItem[]): LiveItem[] {
  return feed.slice(0, 5).map((item) => ({
    actor: item.agentId,
    action: "posted",
    target: item.excerpt || "a new dispatch",
    age: formatRelativeTime(item.createdAt),
  }));
}

export function formatRelativeTime(value: string): string {
  const timestamp = new Date(value).getTime();
  if (Number.isNaN(timestamp)) return "Unknown time";

  const diffSeconds = Math.round((timestamp - Date.now()) / 1000);
  const formatter = new Intl.RelativeTimeFormat("en", { numeric: "auto" });
  const ranges = [
    { unit: "day", seconds: 86_400 },
    { unit: "hour", seconds: 3_600 },
    { unit: "minute", seconds: 60 },
    { unit: "second", seconds: 1 },
  ] as const;

  for (const range of ranges) {
    if (Math.abs(diffSeconds) >= range.seconds || range.unit === "second") {
      return formatter.format(
        Math.trunc(diffSeconds / range.seconds),
        range.unit,
      );
    }
  }

  return "just now";
}

export function shorten(value: string): string {
  if (value.length < 12) return value;
  return `${value.slice(0, 6)}...${value.slice(-4)}`;
}

function badgeFromID(value: string): string {
  const match = value.trim().match(/[a-z0-9]/i);
  return match ? match[0].toUpperCase() : "A";
}

function deriveCategory(agent: Agent, posts: Post[]): string {
  const corpus = [agent.personalitySummary, ...posts.map((post) => post.text)]
    .join(" ")
    .toLowerCase();

  if (matchesAny(corpus, ["finance", "macro", "market", "liquidity", "capital", "defi", "trading"])) {
    return "Finance";
  }
  if (matchesAny(corpus, ["builder", "build", "release", "launch", "protocol", "infra", "developer"])) {
    return "Builder";
  }
  if (matchesAny(corpus, ["research", "thesis", "analysis", "signal", "editor", "policy"])) {
    return "Research";
  }
  if (matchesAny(corpus, ["art", "visual", "image", "illustration", "design", "creative"])) {
    return "Art";
  }
  if (matchesAny(corpus, ["sarcastic", "meme", "funny", "contrarian", "chaos", "glitch"])) {
    return "Persona";
  }
  return "General";
}

function matchesAny(corpus: string, keywords: string[]): boolean {
  return keywords.some((keyword) => corpus.includes(keyword));
}

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// Generates the star history chart embedded in README.md.
//
// Renders two SVGs, one per colour scheme, so the README's <picture> element
// can switch between them without relying on GitHub preserving CSS inside an
// SVG. Run with `deno task star-history`.
//
// Fails loudly rather than writing a partial chart: if the API misbehaves the
// previously committed SVGs stay in place and keep rendering.

const REPO = Deno.env.get("STAR_HISTORY_REPO") ?? "tristanisham/zvm";
const OUT_DIR = ".github/assets";
const MAX_POINTS = 200;

interface Theme {
  name: string;
  background: string;
  text: string;
  muted: string;
  grid: string;
  line: string;
  fill: string;
}

const THEMES: Theme[] = [
  {
    name: "light",
    background: "#ffffff",
    text: "#1f2328",
    muted: "#59636e",
    grid: "#d1d9e0",
    line: "#0969da",
    fill: "#0969da22",
  },
  {
    name: "dark",
    background: "#0d1117",
    text: "#e6edf3",
    muted: "#9198a1",
    grid: "#30363d",
    line: "#58a6ff",
    fill: "#58a6ff22",
  },
];

/** Fetches every stargazer timestamp, oldest first. */
async function fetchStarTimestamps(repo: string): Promise<Date[]> {
  const token = Deno.env.get("GITHUB_TOKEN");
  const headers: HeadersInit = {
    // Without this Accept header the API omits `starred_at`.
    "Accept": "application/vnd.github.star+json",
    "User-Agent": "zvm-star-history",
    "X-GitHub-Api-Version": "2022-11-28",
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const dates: Date[] = [];
  const perPage = 100;

  // The stargazers endpoint caps out at 400 pages; the guard is a runaway
  // backstop, not an expected limit.
  for (let page = 1; page <= 400; page++) {
    const url =
      `https://api.github.com/repos/${repo}/stargazers?per_page=${perPage}&page=${page}`;
    const response = await fetch(url, { headers });

    if (!response.ok) {
      throw new Error(
        `GitHub API returned ${response.status} ${response.statusText} for page ${page}: ${await response
          .text()}`,
      );
    }

    const batch = await response.json();
    if (!Array.isArray(batch)) {
      throw new Error(`Expected an array of stargazers, got ${typeof batch}`);
    }

    for (const entry of batch) {
      if (typeof entry?.starred_at !== "string") {
        throw new Error(
          "Stargazer entry is missing starred_at; the Accept header may have been rejected",
        );
      }
      dates.push(new Date(entry.starred_at));
    }

    if (batch.length < perPage) break;
  }

  if (dates.length === 0) throw new Error(`No stargazers found for ${repo}`);

  dates.sort((a, b) => a.getTime() - b.getTime());
  return dates;
}

interface Point {
  time: number;
  stars: number;
}

/**
 * Turns timestamps into a cumulative series capped at `MAX_POINTS`, always
 * keeping the first and last samples so the chart's endpoints stay exact.
 */
function buildSeries(dates: Date[]): Point[] {
  const total = dates.length;
  const step = Math.max(1, Math.ceil(total / MAX_POINTS));
  const points: Point[] = [];

  for (let i = 0; i < total; i += step) {
    points.push({ time: dates[i].getTime(), stars: i + 1 });
  }

  const last = { time: dates[total - 1].getTime(), stars: total };
  if (points[points.length - 1].stars !== last.stars) points.push(last);

  return points;
}

/** Picks a round tick interval so the y-axis lands on human numbers. */
function niceTicks(max: number, count: number): number[] {
  const rawStep = max / count;
  const magnitude = 10 ** Math.floor(Math.log10(rawStep));
  const normalized = rawStep / magnitude;
  // 2.5 matters: without it a max like 1039 rounds straight from step 2 to
  // step 5, collapsing the axis to three ticks.
  const step = (normalized <= 1
    ? 1
    : normalized <= 2
    ? 2
    : normalized <= 2.5
    ? 2.5
    : normalized <= 5
    ? 5
    : 10) * magnitude;

  const ticks: number[] = [];
  for (let value = 0; value <= max + step / 2; value += step) {
    ticks.push(Math.round(value));
  }
  return ticks;
}

function formatMonth(time: number): string {
  return new Date(time).toLocaleDateString("en-US", {
    month: "short",
    year: "numeric",
    timeZone: "UTC",
  });
}

function escapeXml(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

function renderSvg(points: Point[], repo: string, theme: Theme): string {
  const width = 800;
  const height = 400;
  const margin = { top: 56, right: 32, bottom: 48, left: 72 };
  const plotWidth = width - margin.left - margin.right;
  const plotHeight = height - margin.top - margin.bottom;

  const minTime = points[0].time;
  const maxTime = points[points.length - 1].time;
  const timeSpan = Math.max(1, maxTime - minTime);
  const maxStars = points[points.length - 1].stars;

  const yTicks = niceTicks(maxStars, 5);
  const yMax = yTicks[yTicks.length - 1];

  const x = (time: number) =>
    margin.left + ((time - minTime) / timeSpan) * plotWidth;
  const y = (stars: number) =>
    margin.top + plotHeight - (stars / yMax) * plotHeight;

  const coords = points.map((p) => `${x(p.time).toFixed(1)},${y(p.stars).toFixed(1)}`);
  const areaPath = [
    `M ${x(minTime).toFixed(1)},${(margin.top + plotHeight).toFixed(1)}`,
    ...coords.map((c) => `L ${c.replace(",", " ")}`),
    `L ${x(maxTime).toFixed(1)},${(margin.top + plotHeight).toFixed(1)}`,
    "Z",
  ].join(" ");

  const gridLines = yTicks.map((tick) => {
    const ty = y(tick).toFixed(1);
    return `    <line x1="${margin.left}" y1="${ty}" x2="${
      margin.left + plotWidth
    }" y2="${ty}" stroke="${theme.grid}" stroke-width="1" />
    <text x="${margin.left - 12}" y="${ty}" fill="${theme.muted}" font-size="12" text-anchor="end" dominant-baseline="middle">${tick}</text>`;
  }).join("\n");

  const xTickCount = 6;
  const xLabels = Array.from({ length: xTickCount }, (_, i) => {
    const time = minTime + (timeSpan * i) / (xTickCount - 1);
    const tx = x(time).toFixed(1);
    const anchor = i === 0 ? "start" : i === xTickCount - 1 ? "end" : "middle";
    return `    <text x="${tx}" y="${
      margin.top + plotHeight + 24
    }" fill="${theme.muted}" font-size="12" text-anchor="${anchor}">${formatMonth(time)}</text>`;
  }).join("\n");

  const font =
    "-apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif";

  return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" font-family="${font}" role="img" aria-label="Star history chart for ${
    escapeXml(repo)
  }">
  <rect width="${width}" height="${height}" fill="${theme.background}" />
  <text x="${margin.left}" y="28" fill="${theme.text}" font-size="18" font-weight="600">Star History</text>
  <text x="${margin.left}" y="47" fill="${theme.muted}" font-size="12">${
    escapeXml(repo)
  } &#183; ${maxStars} stars</text>
  <g>
${gridLines}
  </g>
  <path d="${areaPath}" fill="${theme.fill}" />
  <polyline points="${
    coords.join(" ")
  }" fill="none" stroke="${theme.line}" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />
  <g>
${xLabels}
  </g>
  <line x1="${margin.left}" y1="${margin.top + plotHeight}" x2="${
    margin.left + plotWidth
  }" y2="${margin.top + plotHeight}" stroke="${theme.grid}" stroke-width="1" />
</svg>
`;
}

const dates = await fetchStarTimestamps(REPO);
const series = buildSeries(dates);

await Deno.mkdir(OUT_DIR, { recursive: true });
for (const theme of THEMES) {
  const path = `${OUT_DIR}/star-history-${theme.name}.svg`;
  await Deno.writeTextFile(path, renderSvg(series, REPO, theme));
  console.log(`wrote ${path}`);
}

console.log(
  `${dates.length} stars, ${series.length} plotted points, ${
    formatMonth(series[0].time)
  } to ${formatMonth(series[series.length - 1].time)}`,
);

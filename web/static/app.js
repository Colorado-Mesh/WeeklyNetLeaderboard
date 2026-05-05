const tooltip = document.createElement("div");
tooltip.className = "observer-tooltip";
document.body.appendChild(tooltip);

let activeBadge = null;
let refreshInFlight = false;

function applyTileAnimations() {
  const tiles = document.querySelectorAll(".tile");
  tiles.forEach((tile, idx) => {
    tile.style.animationDelay = `${(idx % 10) * 0.13}s`;
  });
}

function placeTooltip(target) {
  const rect = target.getBoundingClientRect();
  const margin = 10;
  const top = Math.max(margin, rect.top - tooltip.offsetHeight - margin);
  const left = Math.min(
    window.innerWidth - tooltip.offsetWidth - margin,
    Math.max(margin, rect.left + rect.width / 2 - tooltip.offsetWidth / 2),
  );
  tooltip.style.top = `${top}px`;
  tooltip.style.left = `${left}px`;
}

function showTooltip(target) {
  const text = target.dataset.tooltip?.trim();
  if (!text) return;
  activeBadge = target;
  tooltip.textContent = text;
  tooltip.classList.add("is-visible");
  placeTooltip(target);
}

function hideTooltip(target) {
  if (!activeBadge || activeBadge !== target) return;
  activeBadge = null;
  tooltip.classList.remove("is-visible");
}

function wireBadgeTooltips() {
  const badges = document.querySelectorAll(".observer-badge[data-tooltip]");
  badges.forEach((badge) => {
    badge.addEventListener("mouseenter", () => showTooltip(badge));
    badge.addEventListener("focus", () => showTooltip(badge));
    badge.addEventListener("mouseleave", () => hideTooltip(badge));
    badge.addEventListener("blur", () => hideTooltip(badge));
  });
}

window.addEventListener("scroll", () => {
  if (activeBadge) placeTooltip(activeBadge);
}, { passive: true });

window.addEventListener("resize", () => {
  if (activeBadge) placeTooltip(activeBadge);
});

async function refreshMainInPlace() {
  if (refreshInFlight) return;
  refreshInFlight = true;
  try {
    const response = await fetch(window.location.href, {
      cache: "no-store",
      headers: { "X-Requested-With": "meshmonday-poll" },
    });
    if (!response.ok) return;

    const html = await response.text();
    const parser = new DOMParser();
    const nextDoc = parser.parseFromString(html, "text/html");
    const nextMain = nextDoc.querySelector("main.page");
    const currentMain = document.querySelector("main.page");
    if (!nextMain || !currentMain) return;

    tooltip.classList.remove("is-visible");
    activeBadge = null;

    currentMain.replaceWith(nextMain);
    if (nextDoc.body?.dataset?.pollSeconds) {
      document.body.dataset.pollSeconds = nextDoc.body.dataset.pollSeconds;
    }
    if (nextDoc.title) {
      document.title = nextDoc.title;
    }
    applyTileAnimations();
    wireBadgeTooltips();
  } catch (_err) {
    // Ignore intermittent polling errors and retry next interval.
  } finally {
    refreshInFlight = false;
  }
}

applyTileAnimations();
wireBadgeTooltips();

const pollSeconds = Number(document.body?.dataset?.pollSeconds ?? 0);
if (Number.isFinite(pollSeconds) && pollSeconds > 0) {
  window.setInterval(() => {
    if (document.visibilityState === "visible") {
      refreshMainInPlace();
    }
  }, pollSeconds * 1000);
}

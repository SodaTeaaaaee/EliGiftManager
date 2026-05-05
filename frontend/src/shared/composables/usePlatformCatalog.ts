import { computed, ref, watch } from "vue";

const STORAGE_KEY = "eligift_platformCatalog";

export interface PlatformEntry {
  name: string;
  type: "member" | "factory";
  notes: string;
}

function loadAll(): PlatformEntry[] {
  if (typeof localStorage === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const arr = JSON.parse(raw);
    if (!Array.isArray(arr)) return [];
    return arr.filter(
      (e): e is PlatformEntry =>
        typeof e.name === "string" &&
        e.name.trim() &&
        (e.type === "member" || e.type === "factory"),
    );
  } catch {
    return [];
  }
}

const platforms = ref<PlatformEntry[]>(loadAll());

watch(
  platforms,
  (v) => {
    if (typeof localStorage !== "undefined") {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(v));
    }
  },
  { deep: true },
);

export function usePlatformCatalog() {
  const platformOptions = computed(() =>
    platforms.value.map((p) => ({ label: p.name, value: p.name, type: p.type })),
  );

  const memberPlatforms = computed(() =>
    platforms.value.filter((p) => p.type === "member"),
  );

  const factoryPlatforms = computed(() =>
    platforms.value.filter((p) => p.type === "factory"),
  );

  function addPlatform(
    name: string,
    type: "member" | "factory",
    notes?: string,
  ): boolean {
    const trimmed = name.trim();
    if (!trimmed || platforms.value.some((p) => p.name === trimmed)) return false;
    platforms.value = [
      ...platforms.value,
      { name: trimmed, type, notes: notes || "" },
    ];
    return true;
  }

  function updatePlatform(
    oldName: string,
    updates: Partial<Omit<PlatformEntry, "name">> & { name?: string },
  ): boolean {
    const idx = platforms.value.findIndex((p) => p.name === oldName);
    if (idx === -1) return false;
    const newName = (updates.name || oldName).trim();
    if (!newName) return false;
    if (
      newName !== oldName &&
      platforms.value.some((p) => p.name === newName)
    ) {
      return false;
    }
    const updated = { ...platforms.value[idx], ...updates, name: newName };
    platforms.value = [
      ...platforms.value.slice(0, idx),
      updated,
      ...platforms.value.slice(idx + 1),
    ];
    return true;
  }

  function removePlatform(name: string): boolean {
    const idx = platforms.value.findIndex((p) => p.name === name);
    if (idx === -1) return false;
    platforms.value = [
      ...platforms.value.slice(0, idx),
      ...platforms.value.slice(idx + 1),
    ];
    return true;
  }

  function getPlatform(name: string): PlatformEntry | undefined {
    return platforms.value.find((p) => p.name === name);
  }

  return {
    platforms,
    platformOptions,
    memberPlatforms,
    factoryPlatforms,
    addPlatform,
    updatePlatform,
    removePlatform,
    getPlatform,
  };
}

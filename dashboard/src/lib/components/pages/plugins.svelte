<script lang="ts">
  import { onMount } from "svelte";
  import { api, type Plugin } from "$lib/api";
  import * as Card from "$lib/components/ui/card/index.js";
  import * as Dialog from "$lib/components/ui/dialog/index.js";
  import { Badge } from "$lib/components/ui/badge/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Skeleton } from "$lib/components/ui/skeleton/index.js";
  import PuzzleIcon from "@lucide/svelte/icons/puzzle";
  import WrenchIcon from "@lucide/svelte/icons/wrench";

  let plugins = $state<Plugin[]>([]);
  let loading = $state(true);
  let selectedPlugin = $state<Plugin | null>(null);
  let showDetail = $state(false);

  onMount(async () => {
    try {
      const resp = await api.getPlugins();
      plugins = resp.plugins || [];
    } catch (e) {
      console.error("Failed to fetch plugins:", e);
    } finally {
      loading = false;
    }
  });

  function runtimeColor(
    rt?: string
  ): "default" | "secondary" | "destructive" | "outline" {
    switch (rt) {
      case "command":
        return "default";
      case "http":
        return "secondary";
      case "mcp":
        return "secondary";
      case "host":
        return "destructive";
      default:
        return "outline";
    }
  }

  function openDetail(plugin: Plugin) {
    selectedPlugin = plugin;
    showDetail = true;
  }
</script>

<div class="space-y-6">
  <h1 class="text-3xl font-bold">Plugins</h1>
  <p class="text-muted-foreground">
    Available plugin packages in the registry. Workers install plugins on demand
    when a profile needs them. Click a plugin for details.
  </p>

  {#if loading}
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {#each Array(4) as _}
        <Card.Root>
          <Card.Content class="p-6">
            <Skeleton class="h-24 w-full" />
          </Card.Content>
        </Card.Root>
      {/each}
    </div>
  {:else if plugins.length === 0}
    <Card.Root>
      <Card.Content
        class="flex flex-col items-center justify-center py-12 text-center"
      >
        <PuzzleIcon class="h-12 w-12 text-muted-foreground mb-4" />
        <p class="text-lg font-medium">No plugins</p>
        <p class="text-sm text-muted-foreground">
          Publish plugins to the registry to see them here
        </p>
      </Card.Content>
    </Card.Root>
  {:else}
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {#each plugins as plugin (plugin.name)}
        <button class="text-left" onclick={() => openDetail(plugin)}>
          <Card.Root
            class="h-full transition-colors hover:border-primary/50 cursor-pointer"
          >
            <Card.Header>
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <PuzzleIcon class="h-4 w-4 text-muted-foreground" />
                  <Card.Title class="text-base">{plugin.name}</Card.Title>
                </div>
                <Badge variant="outline" class="text-[10px] font-mono"
                  >v{plugin.version}</Badge
                >
              </div>
              {#if plugin.description}
                <Card.Description>{plugin.description}</Card.Description>
              {/if}
            </Card.Header>
            <Card.Content class="space-y-3">
              <div class="flex flex-wrap gap-2">
                {#if plugin.runtime}
                  <Badge
                    variant={runtimeColor(plugin.runtime)}
                    class="text-[10px]">{plugin.runtime}</Badge
                  >
                {/if}
                {#if plugin.category}
                  <Badge variant="outline" class="text-[10px]"
                    >{plugin.category}</Badge
                  >
                {/if}
              </div>
              {#if plugin.tools?.length}
                <div>
                  <p class="text-xs text-muted-foreground mb-1">Tools</p>
                  <div class="flex flex-wrap gap-1">
                    {#each plugin.tools as t}
                      <Badge
                        variant="outline"
                        class="font-mono text-[10px]">{t}</Badge
                      >
                    {/each}
                  </div>
                </div>
              {/if}
            </Card.Content>
          </Card.Root>
        </button>
      {/each}
    </div>
  {/if}
</div>

<!-- Plugin detail dialog -->
<Dialog.Root bind:open={showDetail}>
  <Dialog.Content class="max-w-lg">
    {#if selectedPlugin}
      <Dialog.Header>
        <div class="flex items-center gap-3">
          <PuzzleIcon class="h-6 w-6 text-primary" />
          <div>
            <Dialog.Title class="text-xl"
              >{selectedPlugin.name}</Dialog.Title
            >
            <Dialog.Description>
              {selectedPlugin.description || "No description"}
            </Dialog.Description>
          </div>
        </div>
      </Dialog.Header>

      <div class="space-y-4 py-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <p class="text-xs text-muted-foreground font-medium uppercase mb-1">
              Version
            </p>
            <Badge variant="outline" class="font-mono"
              >v{selectedPlugin.version}</Badge
            >
          </div>
          <div>
            <p class="text-xs text-muted-foreground font-medium uppercase mb-1">
              Runtime
            </p>
            <Badge variant={runtimeColor(selectedPlugin.runtime)}
              >{selectedPlugin.runtime || "—"}</Badge
            >
          </div>
          <div>
            <p class="text-xs text-muted-foreground font-medium uppercase mb-1">
              Category
            </p>
            <Badge variant="outline">{selectedPlugin.category || "—"}</Badge>
          </div>
          <div>
            <p class="text-xs text-muted-foreground font-medium uppercase mb-1">
              Source
            </p>
            <Badge variant="outline">{selectedPlugin.source}</Badge>
          </div>
        </div>

        {#if selectedPlugin.tools?.length}
          <div>
            <p class="text-xs text-muted-foreground font-medium uppercase mb-2">
              Contributed Tools
            </p>
            <div class="flex flex-wrap gap-1.5">
              {#each selectedPlugin.tools as t}
                <Badge variant="secondary" class="font-mono">
                  <WrenchIcon class="h-3 w-3 mr-1" />
                  {t}
                </Badge>
              {/each}
            </div>
          </div>
        {/if}

        {#if selectedPlugin.dependencies?.length}
          <div>
            <p class="text-xs text-muted-foreground font-medium uppercase mb-2">
              Dependencies
            </p>
            <div class="flex flex-wrap gap-1.5">
              {#each selectedPlugin.dependencies as dep}
                <Badge variant="outline">{dep}</Badge>
              {/each}
            </div>
          </div>
        {/if}
      </div>

      <Dialog.Footer>
        <Button variant="outline" onclick={() => (showDetail = false)}
          >Close</Button
        >
      </Dialog.Footer>
    {/if}
  </Dialog.Content>
</Dialog.Root>

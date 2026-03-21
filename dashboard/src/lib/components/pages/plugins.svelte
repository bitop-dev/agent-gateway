<script lang="ts">
  import { onMount } from "svelte";
  import { api, type Plugin } from "$lib/api";
  import * as Card from "$lib/components/ui/card/index.js";
  import { Badge } from "$lib/components/ui/badge/index.js";
  import { Skeleton } from "$lib/components/ui/skeleton/index.js";
  import PuzzleIcon from "@lucide/svelte/icons/puzzle";

  let plugins = $state<Plugin[]>([]);
  let loading = $state(true);

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
</script>

<div class="space-y-6">
  <h1 class="text-3xl font-bold">Plugins</h1>
  <p class="text-muted-foreground">
    Available plugin packages in the registry. Workers install plugins on demand
    when a profile needs them.
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
        <Card.Root>
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
          <Card.Content>
            <div class="flex flex-wrap gap-2">
              {#if plugin.runtime}
                <Badge variant={runtimeColor(plugin.runtime)} class="text-[10px]"
                  >{plugin.runtime}</Badge
                >
              {/if}
              {#if plugin.category}
                <Badge variant="outline" class="text-[10px]"
                  >{plugin.category}</Badge
                >
              {/if}
              <Badge variant="outline" class="text-[10px]"
                >{plugin.source}</Badge
              >
            </div>
          </Card.Content>
        </Card.Root>
      {/each}
    </div>
  {/if}
</div>

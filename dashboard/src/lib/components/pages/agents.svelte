<script lang="ts">
  import { onMount } from "svelte";
  import { api, type Agent } from "$lib/api";
  import * as Card from "$lib/components/ui/card/index.js";
  import { Badge } from "$lib/components/ui/badge/index.js";
  import { Skeleton } from "$lib/components/ui/skeleton/index.js";
  import BotIcon from "@lucide/svelte/icons/bot";

  let agents = $state<Agent[]>([]);
  let loading = $state(true);

  onMount(async () => {
    try {
      const resp = await api.getAgents();
      agents = resp.agents || [];
    } catch (e) {
      console.error("Failed to fetch agents:", e);
    } finally {
      loading = false;
    }
  });
</script>

<div class="space-y-6">
  <h1 class="text-3xl font-bold">Agents</h1>
  <p class="text-muted-foreground">
    Available agent profiles across workers and registry.
  </p>

  {#if loading}
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {#each Array(3) as _}
        <Card.Root>
          <Card.Content class="p-6">
            <Skeleton class="h-24 w-full" />
          </Card.Content>
        </Card.Root>
      {/each}
    </div>
  {:else if agents.length === 0}
    <Card.Root>
      <Card.Content class="flex flex-col items-center justify-center py-12 text-center">
        <BotIcon class="h-12 w-12 text-muted-foreground mb-4" />
        <p class="text-lg font-medium">No agents discovered</p>
        <p class="text-sm text-muted-foreground">
          Publish profiles to the registry to see them here
        </p>
      </Card.Content>
    </Card.Root>
  {:else}
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {#each agents as agent (agent.name)}
        <Card.Root>
          <Card.Header>
            <div class="flex items-center justify-between">
              <Card.Title>{agent.name}</Card.Title>
              <Badge variant="outline" class="text-[10px]">{agent.source}</Badge>
            </div>
            {#if agent.description}
              <Card.Description>{agent.description}</Card.Description>
            {/if}
          </Card.Header>
          <Card.Content class="space-y-3">
            {#if agent.capabilities?.length}
              <div>
                <p class="text-xs text-muted-foreground mb-1">Capabilities</p>
                <div class="flex flex-wrap gap-1">
                  {#each agent.capabilities as cap}
                    <Badge variant="secondary" class="text-[10px]">{cap}</Badge>
                  {/each}
                </div>
              </div>
            {/if}
            {#if agent.accepts}
              <div>
                <p class="text-xs text-muted-foreground">Accepts</p>
                <p class="text-sm">{agent.accepts}</p>
              </div>
            {/if}
            {#if agent.returns}
              <div>
                <p class="text-xs text-muted-foreground">Returns</p>
                <p class="text-sm">{agent.returns}</p>
              </div>
            {/if}
          </Card.Content>
        </Card.Root>
      {/each}
    </div>
  {/if}
</div>

<script lang="ts">
  import { onMount } from "svelte";
  import { api, type Agent } from "$lib/api";
  import * as Card from "$lib/components/ui/card/index.js";
  import * as Dialog from "$lib/components/ui/dialog/index.js";
  import { Badge } from "$lib/components/ui/badge/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Skeleton } from "$lib/components/ui/skeleton/index.js";
  import { navigate } from "$lib/router";
  import BotIcon from "@lucide/svelte/icons/bot";
  import SendIcon from "@lucide/svelte/icons/send";

  let agents = $state<Agent[]>([]);
  let loading = $state(true);
  let selectedAgent = $state<Agent | null>(null);
  let showDetail = $state(false);

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

  function openDetail(agent: Agent) {
    selectedAgent = agent;
    showDetail = true;
  }

  function submitTask(name: string) {
    showDetail = false;
    navigate(`/tasks/new?profile=${encodeURIComponent(name)}`);
  }
</script>

<div class="space-y-6">
  <h1 class="text-3xl font-bold">Agents</h1>
  <p class="text-muted-foreground">
    Available agent profiles across workers and registry. Click an agent to see
    details.
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
      <Card.Content
        class="flex flex-col items-center justify-center py-12 text-center"
      >
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
        <button
          class="text-left"
          onclick={() => openDetail(agent)}
        >
          <Card.Root
            class="h-full transition-colors hover:border-primary/50 cursor-pointer"
          >
            <Card.Header>
              <div class="flex items-center justify-between">
                <Card.Title>{agent.name}</Card.Title>
                <Badge variant="outline" class="text-[10px]"
                  >{agent.source}</Badge
                >
              </div>
              {#if agent.description}
                <Card.Description>{agent.description}</Card.Description>
              {/if}
            </Card.Header>
            <Card.Content class="space-y-3">
              {#if agent.capabilities?.length}
                <div>
                  <div class="flex flex-wrap gap-1">
                    {#each agent.capabilities as cap}
                      <Badge variant="secondary" class="text-[10px]"
                        >{cap}</Badge
                      >
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
        </button>
      {/each}
    </div>
  {/if}
</div>

<!-- Agent detail dialog -->
<Dialog.Root bind:open={showDetail}>
  <Dialog.Content class="max-w-lg">
    {#if selectedAgent}
      <Dialog.Header>
        <div class="flex items-center gap-3">
          <BotIcon class="h-6 w-6 text-primary" />
          <div>
            <Dialog.Title class="text-xl"
              >{selectedAgent.name}</Dialog.Title
            >
            <Dialog.Description>
              {selectedAgent.description || "No description"}
            </Dialog.Description>
          </div>
        </div>
      </Dialog.Header>

      <div class="space-y-4 py-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <p class="text-xs text-muted-foreground font-medium uppercase mb-1">
              Source
            </p>
            <Badge variant="outline">{selectedAgent.source}</Badge>
          </div>
          {#if selectedAgent.provider || selectedAgent.model}
            <div>
              <p class="text-xs text-muted-foreground font-medium uppercase mb-1">
                Provider / Model
              </p>
              <p class="text-sm font-mono">
                {selectedAgent.provider || "openai"} / {selectedAgent.model || "—"}
              </p>
            </div>
          {/if}
          {#if selectedAgent.mode}
            <div>
              <p class="text-xs text-muted-foreground font-medium uppercase mb-1">
                Mode
              </p>
              <Badge variant="secondary">{selectedAgent.mode}</Badge>
            </div>
          {/if}
          {#if selectedAgent.extends}
            <div>
              <p class="text-xs text-muted-foreground font-medium uppercase mb-1">
                Extends
              </p>
              <Badge variant="outline">{selectedAgent.extends}</Badge>
            </div>
          {/if}
        </div>

        {#if selectedAgent.capabilities?.length}
          <div>
            <p class="text-xs text-muted-foreground font-medium uppercase mb-2">
              Capabilities
            </p>
            <div class="flex flex-wrap gap-1.5">
              {#each selectedAgent.capabilities as cap}
                <Badge variant="secondary">{cap}</Badge>
              {/each}
            </div>
          </div>
        {/if}

        {#if selectedAgent.tools?.length}
          <div>
            <p class="text-xs text-muted-foreground font-medium uppercase mb-2">
              Tools
            </p>
            <div class="flex flex-wrap gap-1.5">
              {#each selectedAgent.tools as t}
                <Badge variant="outline" class="font-mono text-[11px]">{t}</Badge>
              {/each}
            </div>
          </div>
        {/if}

        {#if selectedAgent.accepts}
          <div>
            <p class="text-xs text-muted-foreground font-medium uppercase mb-1">
              Accepts
            </p>
            <p class="text-sm">{selectedAgent.accepts}</p>
          </div>
        {/if}

        {#if selectedAgent.returns}
          <div>
            <p class="text-xs text-muted-foreground font-medium uppercase mb-1">
              Returns
            </p>
            <p class="text-sm">{selectedAgent.returns}</p>
          </div>
        {/if}
      </div>

      <Dialog.Footer>
        <Button variant="outline" onclick={() => (showDetail = false)}
          >Close</Button
        >
        <Button onclick={() => submitTask(selectedAgent?.name || "")}>
          <SendIcon class="h-4 w-4 mr-1" />
          Submit Task
        </Button>
      </Dialog.Footer>
    {/if}
  </Dialog.Content>
</Dialog.Root>

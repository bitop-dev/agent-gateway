<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { api, type Worker } from "$lib/api";
  import * as Card from "$lib/components/ui/card/index.js";
  import * as Table from "$lib/components/ui/table/index.js";
  import { Badge } from "$lib/components/ui/badge/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Skeleton } from "$lib/components/ui/skeleton/index.js";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import ServerIcon from "@lucide/svelte/icons/server";

  let workers = $state<Worker[]>([]);
  let loading = $state(true);
  let interval: ReturnType<typeof setInterval>;

  async function refresh() {
    try {
      const resp = await api.getWorkers();
      workers = resp.workers || [];
    } catch (e) {
      console.error("Failed to fetch workers:", e);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    refresh();
    interval = setInterval(refresh, 10000);
  });

  onDestroy(() => clearInterval(interval));

  function timeAgo(ts: string): string {
    if (!ts) return "";
    const diff = Date.now() - new Date(ts).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return "just now";
    if (mins < 60) return `${mins}m ago`;
    return `${Math.floor(mins / 60)}h ago`;
  }
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-3">
      <h1 class="text-3xl font-bold">Workers</h1>
      <Badge variant="secondary">{workers.length} active</Badge>
    </div>
    <Button variant="outline" size="sm" onclick={refresh}>
      <RefreshCwIcon class="h-4 w-4" />
    </Button>
  </div>

  {#if loading}
    <div class="grid gap-4 md:grid-cols-3">
      {#each Array(3) as _}
        <Card.Root>
          <Card.Content class="p-6">
            <Skeleton class="h-20 w-full" />
          </Card.Content>
        </Card.Root>
      {/each}
    </div>
  {:else if workers.length === 0}
    <Card.Root>
      <Card.Content class="flex flex-col items-center justify-center py-12 text-center">
        <ServerIcon class="h-12 w-12 text-muted-foreground mb-4" />
        <p class="text-lg font-medium">No workers connected</p>
        <p class="text-sm text-muted-foreground">
          Workers register automatically when they start
        </p>
      </Card.Content>
    </Card.Root>
  {:else}
    <!-- Worker cards -->
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {#each workers as worker (worker.url)}
        <Card.Root>
          <Card.Header class="pb-3">
            <div class="flex items-center justify-between">
              <Card.Title class="text-sm font-mono">{worker.url}</Card.Title>
              <Badge
                variant={worker.status === "idle" ? "default" : "secondary"}
              >
                {worker.status || "idle"}
              </Badge>
            </div>
          </Card.Header>
          <Card.Content class="space-y-2 text-sm">
            {#if worker.currentTask}
              <div>
                <span class="text-muted-foreground">Current:</span>
                <span class="font-mono text-xs ml-1"
                  >{worker.currentTask}</span
                >
              </div>
            {/if}
            <div>
              <span class="text-muted-foreground">Completed:</span>
              <span class="ml-1">{worker.completedTasks ?? 0}</span>
            </div>
            {#if worker.profiles?.length}
              <div>
                <span class="text-muted-foreground">Profiles:</span>
                <div class="flex flex-wrap gap-1 mt-1">
                  {#each worker.profiles as p}
                    <Badge variant="outline" class="text-[10px]">{p}</Badge>
                  {/each}
                </div>
              </div>
            {/if}
            <div class="text-xs text-muted-foreground">
              Last seen: {timeAgo(worker.lastSeen)}
            </div>
          </Card.Content>
        </Card.Root>
      {/each}
    </div>
  {/if}
</div>

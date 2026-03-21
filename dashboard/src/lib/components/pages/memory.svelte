<script lang="ts">
  import { onMount } from "svelte";
  import { api, type Agent, type MemoryEntry } from "$lib/api";
  import * as Card from "$lib/components/ui/card/index.js";
  import * as Table from "$lib/components/ui/table/index.js";
  import * as Dialog from "$lib/components/ui/dialog/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Input } from "$lib/components/ui/input/index.js";
  import { Skeleton } from "$lib/components/ui/skeleton/index.js";
  import { Badge } from "$lib/components/ui/badge/index.js";
  import BrainIcon from "@lucide/svelte/icons/brain";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";

  let profiles = $state<string[]>([]);
  let selectedProfile = $state("");
  let memories = $state<MemoryEntry[]>([]);
  let loading = $state(true);
  let memLoading = $state(false);

  // Add dialog
  let showAdd = $state(false);
  let newKey = $state("");
  let newValue = $state("");

  onMount(async () => {
    try {
      const resp = await api.getAgents();
      profiles = (resp.agents || []).map((a) => a.name);
      if (profiles.length > 0) {
        selectedProfile = profiles[0];
        await loadMemory();
      }
    } catch (e) {
      console.error("Failed to load:", e);
    } finally {
      loading = false;
    }
  });

  async function loadMemory() {
    if (!selectedProfile) return;
    memLoading = true;
    try {
      const resp = await api.getMemory(selectedProfile);
      memories = resp.entries || [];
    } catch (e) {
      console.error("Failed to load memory:", e);
      memories = [];
    } finally {
      memLoading = false;
    }
  }

  async function addEntry() {
    if (!newKey.trim() || !newValue.trim()) return;
    try {
      await api.setMemory(selectedProfile, newKey.trim(), newValue.trim());
      newKey = "";
      newValue = "";
      showAdd = false;
      await loadMemory();
    } catch (e) {
      console.error("Failed to add memory:", e);
    }
  }

  async function deleteEntry(key: string) {
    try {
      await api.deleteMemory(selectedProfile, key);
      await loadMemory();
    } catch (e) {
      console.error("Failed to delete memory:", e);
    }
  }

  async function clearAll() {
    if (!confirm("Delete all memories for " + selectedProfile + "?")) return;
    try {
      await api.deleteMemory(selectedProfile);
      await loadMemory();
    } catch (e) {
      console.error("Failed to clear memory:", e);
    }
  }

  function timeAgo(ts: string): string {
    if (!ts) return "";
    const diff = Date.now() - new Date(ts).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return "just now";
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    return `${Math.floor(hours / 24)}d ago`;
  }
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-3xl font-bold">Agent Memory</h1>
    <div class="flex gap-2">
      <Button variant="outline" size="sm" onclick={loadMemory}>
        <RefreshCwIcon class="h-4 w-4" />
      </Button>
      <Button
        size="sm"
        onclick={() => (showAdd = true)}
        disabled={!selectedProfile}
      >
        <PlusIcon class="h-4 w-4 mr-1" />
        Add
      </Button>
    </div>
  </div>

  <!-- Profile picker -->
  <div class="flex items-center gap-3">
    <label for="mem-profile" class="text-sm font-medium">Profile:</label>
    <select
      id="mem-profile"
      class="flex h-9 rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
      bind:value={selectedProfile}
      onchange={() => loadMemory()}
    >
      {#each profiles as p}
        <option value={p}>{p}</option>
      {/each}
    </select>
    {#if memories.length > 0}
      <Badge variant="secondary">{memories.length} entries</Badge>
      <Button variant="ghost" size="sm" class="text-destructive" onclick={clearAll}>
        <Trash2Icon class="h-4 w-4 mr-1" />
        Clear All
      </Button>
    {/if}
  </div>

  <Card.Root>
    <Card.Content class="p-0">
      {#if loading || memLoading}
        <div class="p-6 space-y-3">
          {#each Array(3) as _}
            <Skeleton class="h-10 w-full" />
          {/each}
        </div>
      {:else if memories.length === 0}
        <div class="flex flex-col items-center justify-center py-12 text-center">
          <BrainIcon class="h-12 w-12 text-muted-foreground mb-4" />
          <p class="text-lg font-medium">No memories</p>
          <p class="text-sm text-muted-foreground">
            Agents store facts here using agent/remember
          </p>
        </div>
      {:else}
        <Table.Root>
          <Table.Header>
            <Table.Row>
              <Table.Head class="w-[200px]">Key</Table.Head>
              <Table.Head>Value</Table.Head>
              <Table.Head class="w-[100px]">Updated</Table.Head>
              <Table.Head class="w-[60px]"></Table.Head>
            </Table.Row>
          </Table.Header>
          <Table.Body>
            {#each memories as mem (mem.key)}
              <Table.Row>
                <Table.Cell class="font-mono text-sm font-medium"
                  >{mem.key}</Table.Cell
                >
                <Table.Cell class="text-sm max-w-[400px] truncate"
                  >{mem.value}</Table.Cell
                >
                <Table.Cell class="text-xs text-muted-foreground"
                  >{timeAgo(mem.updatedAt)}</Table.Cell
                >
                <Table.Cell>
                  <Button
                    variant="ghost"
                    size="sm"
                    class="text-destructive h-7 w-7 p-0"
                    onclick={() => deleteEntry(mem.key)}
                  >
                    <Trash2Icon class="h-3 w-3" />
                  </Button>
                </Table.Cell>
              </Table.Row>
            {/each}
          </Table.Body>
        </Table.Root>
      {/if}
    </Card.Content>
  </Card.Root>
</div>

<!-- Add dialog -->
<Dialog.Root bind:open={showAdd}>
  <Dialog.Content>
    <Dialog.Header>
      <Dialog.Title>Add Memory Entry</Dialog.Title>
      <Dialog.Description>
        Store a key-value fact for {selectedProfile}
      </Dialog.Description>
    </Dialog.Header>
    <div class="space-y-4 py-4">
      <div>
        <label for="mem-key" class="text-sm font-medium">Key</label>
        <Input
          id="mem-key"
          placeholder="e.g. last_report_date"
          bind:value={newKey}
        />
      </div>
      <div>
        <label for="mem-value" class="text-sm font-medium">Value</label>
        <Input id="mem-value" placeholder="e.g. 2026-03-21" bind:value={newValue} />
      </div>
    </div>
    <Dialog.Footer>
      <Button variant="outline" onclick={() => (showAdd = false)}>Cancel</Button>
      <Button onclick={addEntry} disabled={!newKey.trim() || !newValue.trim()}>
        Save
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

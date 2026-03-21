<script lang="ts">
  import { onMount } from "svelte";
  import { api, type Agent } from "$lib/api";
  import * as Card from "$lib/components/ui/card/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Textarea } from "$lib/components/ui/textarea/index.js";
  import { Badge } from "$lib/components/ui/badge/index.js";
  import { navigate } from "$lib/router";
  import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
  import SendIcon from "@lucide/svelte/icons/send";
  import LoaderIcon from "@lucide/svelte/icons/loader";

  let agents = $state<Agent[]>([]);
  let selectedProfile = $state("");
  let taskText = $state("");
  let asyncMode = $state(false);
  let submitting = $state(false);
  let result = $state<{ taskId?: string; error?: string; result?: string } | null>(
    null
  );

  onMount(async () => {
    try {
      const resp = await api.getAgents();
      agents = resp.agents || [];
      if (agents.length > 0) selectedProfile = agents[0].name;
    } catch (e) {
      console.error("Failed to fetch agents:", e);
    }
  });

  async function submit() {
    if (!selectedProfile || !taskText.trim()) return;
    submitting = true;
    result = null;
    try {
      const resp = await api.submitTask(
        selectedProfile,
        taskText.trim(),
        asyncMode
      );
      result = resp;
      if (asyncMode && resp.taskId) {
        navigate(`/tasks/${resp.taskId}`);
      }
    } catch (e: any) {
      result = { error: e.message };
    } finally {
      submitting = false;
    }
  }

  let selectedAgent = $derived(agents.find((a) => a.name === selectedProfile));
</script>

<div class="space-y-6">
  <div class="flex items-center gap-4">
    <Button variant="ghost" size="sm" onclick={() => navigate("/tasks")}>
      <ArrowLeftIcon class="h-4 w-4 mr-1" />
      Back
    </Button>
    <h1 class="text-3xl font-bold">New Task</h1>
  </div>

  <div class="grid gap-6 lg:grid-cols-3">
    <!-- Submit form -->
    <div class="lg:col-span-2 space-y-4">
      <Card.Root>
        <Card.Header>
          <Card.Title>Submit a task</Card.Title>
        </Card.Header>
        <Card.Content class="space-y-4">
          <div>
            <label for="profile" class="text-sm font-medium mb-2 block"
              >Profile</label
            >
            <select
              id="profile"
              class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              bind:value={selectedProfile}
            >
              {#each agents as agent (agent.name)}
                <option value={agent.name}>{agent.name}</option>
              {/each}
            </select>
          </div>

          <div>
            <label for="task" class="text-sm font-medium mb-2 block"
              >Task</label
            >
            <Textarea
              id="task"
              placeholder="Describe what you want the agent to do..."
              rows={5}
              bind:value={taskText}
            />
          </div>

          <div class="flex items-center gap-4">
            <label class="flex items-center gap-2 text-sm">
              <input type="checkbox" bind:checked={asyncMode} />
              Async (return immediately)
            </label>
          </div>

          <Button
            class="w-full"
            disabled={!selectedProfile || !taskText.trim() || submitting}
            onclick={submit}
          >
            {#if submitting}
              <LoaderIcon class="h-4 w-4 mr-2 animate-spin" />
              Running...
            {:else}
              <SendIcon class="h-4 w-4 mr-2" />
              Submit
            {/if}
          </Button>
        </Card.Content>
      </Card.Root>

      <!-- Result -->
      {#if result}
        <Card.Root>
          <Card.Header>
            <Card.Title>{result.error ? "Error" : "Result"}</Card.Title>
          </Card.Header>
          <Card.Content>
            {#if result.error}
              <div class="rounded-md bg-destructive/10 p-4 text-sm text-destructive whitespace-pre-wrap">
                {result.error}
              </div>
            {:else if result.taskId}
              <div class="space-y-2">
                <p class="text-sm">
                  Task <button
                    class="font-mono text-primary underline"
                    onclick={() => navigate(`/tasks/${result?.taskId}`)}
                    >{result.taskId}</button
                  >
                </p>
                {#if result.result}
                  <div class="rounded-md bg-muted p-4 text-sm whitespace-pre-wrap max-h-[400px] overflow-y-auto">
                    {result.result}
                  </div>
                {/if}
              </div>
            {/if}
          </Card.Content>
        </Card.Root>
      {/if}
    </div>

    <!-- Available agents -->
    <div>
      <Card.Root>
        <Card.Header>
          <Card.Title>Available Agents</Card.Title>
        </Card.Header>
        <Card.Content>
          {#if agents.length === 0}
            <p class="text-sm text-muted-foreground">Loading agents...</p>
          {:else}
            <div class="space-y-3">
              {#each agents as agent (agent.name)}
                <button
                  class="w-full text-left rounded-md p-3 border {selectedProfile ===
                  agent.name
                    ? 'border-primary bg-accent'
                    : 'border-border'} hover:bg-accent transition-colors"
                  onclick={() => (selectedProfile = agent.name)}
                >
                  <p class="font-medium text-sm">{agent.name}</p>
                  {#if agent.description}
                    <p class="text-xs text-muted-foreground mt-1">
                      {agent.description}
                    </p>
                  {/if}
                  {#if agent.capabilities?.length}
                    <div class="flex flex-wrap gap-1 mt-2">
                      {#each agent.capabilities as cap}
                        <Badge variant="outline" class="text-[10px]"
                          >{cap}</Badge
                        >
                      {/each}
                    </div>
                  {/if}
                </button>
              {/each}
            </div>
          {/if}
        </Card.Content>
      </Card.Root>

      {#if selectedAgent}
        <Card.Root class="mt-4">
          <Card.Header>
            <Card.Title class="text-sm">{selectedAgent.name}</Card.Title>
          </Card.Header>
          <Card.Content class="text-sm space-y-2">
            {#if selectedAgent.accepts}
              <div>
                <span class="text-muted-foreground">Accepts:</span>
                {selectedAgent.accepts}
              </div>
            {/if}
            {#if selectedAgent.returns}
              <div>
                <span class="text-muted-foreground">Returns:</span>
                {selectedAgent.returns}
              </div>
            {/if}
          </Card.Content>
        </Card.Root>
      {/if}
    </div>
  </div>
</div>

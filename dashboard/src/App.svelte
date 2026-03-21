<script lang="ts">
  import "./app.css";
  import { ModeWatcher } from "mode-watcher";
  import { toggleMode } from "mode-watcher";
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Input } from "$lib/components/ui/input/index.js";
  import AppSidebar from "$lib/components/app-sidebar.svelte";
  import { currentPath, matchRoute } from "$lib/router";
  import { apiKey, addEvent } from "$lib/stores";
  import { api } from "$lib/api";
  import { eventStream } from "$lib/sse";
  import SunIcon from "@lucide/svelte/icons/sun";
  import MoonIcon from "@lucide/svelte/icons/moon";
  import KeyIcon from "@lucide/svelte/icons/key-round";

  // Pages
  import Overview from "$lib/components/pages/overview.svelte";
  import Tasks from "$lib/components/pages/tasks.svelte";
  import TaskDetail from "$lib/components/pages/task-detail.svelte";
  import TaskNew from "$lib/components/pages/task-new.svelte";
  import Workers from "$lib/components/pages/workers.svelte";
  import Agents from "$lib/components/pages/agents.svelte";
  import Plugins from "$lib/components/pages/plugins.svelte";
  import Costs from "$lib/components/pages/costs.svelte";
  import Memory from "$lib/components/pages/memory.svelte";
  import Webhooks from "$lib/components/pages/webhooks.svelte";
  import Schedules from "$lib/components/pages/schedules.svelte";

  let showKeyInput = $state(false);
  let keyInput = $state($apiKey);

  // Sync API key to api client and SSE
  $effect(() => {
    api.setKey($apiKey);
    if ($apiKey) {
      eventStream.connect($apiKey);
    } else {
      eventStream.disconnect();
    }
  });

  // SSE events → store
  eventStream.onEvent((event) => {
    addEvent(event);
  });

  function saveKey() {
    apiKey.set(keyInput);
    showKeyInput = false;
  }
</script>

<ModeWatcher defaultMode="dark" />

<Sidebar.Provider>
  <AppSidebar />
  <main class="flex-1 overflow-auto">
    <!-- Top bar -->
    <div
      class="sticky top-0 z-10 flex items-center justify-between border-b bg-background/95 px-6 py-3 backdrop-blur supports-[backdrop-filter]:bg-background/60"
    >
      <div class="flex items-center gap-2">
        <Sidebar.Trigger />
      </div>
      <div class="flex items-center gap-2">
        {#if showKeyInput}
          <div class="flex items-center gap-2">
            <Input
              type="password"
              placeholder="API Key"
              class="h-8 w-48"
              bind:value={keyInput}
              onkeydown={(e: KeyboardEvent) => e.key === "Enter" && saveKey()}
            />
            <Button size="sm" variant="default" onclick={saveKey}>Save</Button>
            <Button
              size="sm"
              variant="ghost"
              onclick={() => (showKeyInput = false)}>Cancel</Button
            >
          </div>
        {:else}
          <Button
            variant="ghost"
            size="sm"
            onclick={() => {
              keyInput = $apiKey;
              showKeyInput = true;
            }}
          >
            <KeyIcon class="h-4 w-4 mr-1" />
            {$apiKey ? "Key ✓" : "Set API Key"}
          </Button>
        {/if}
        <Button variant="ghost" size="icon" class="h-8 w-8" onclick={toggleMode}>
          <SunIcon
            class="h-4 w-4 scale-100 rotate-0 transition-all dark:scale-0 dark:-rotate-90"
          />
          <MoonIcon
            class="absolute h-4 w-4 scale-0 rotate-90 transition-all dark:scale-100 dark:rotate-0"
          />
        </Button>
      </div>
    </div>

    <!-- Page content -->
    <div class="p-6">
      {#if $currentPath === "/"}
        <Overview />
      {:else if $currentPath === "/tasks"}
        <Tasks />
      {:else if $currentPath === "/tasks/new"}
        <TaskNew />
      {:else if matchRoute("/tasks/:id", $currentPath)}
        <TaskDetail taskId={matchRoute("/tasks/:id", $currentPath)?.id ?? ""} />
      {:else if $currentPath === "/workers"}
        <Workers />
      {:else if $currentPath === "/agents"}
        <Agents />
      {:else if $currentPath === "/plugins"}
        <Plugins />
      {:else if $currentPath === "/costs"}
        <Costs />
      {:else if $currentPath === "/memory"}
        <Memory />
      {:else if $currentPath === "/webhooks"}
        <Webhooks />
      {:else if $currentPath === "/schedules"}
        <Schedules />
      {:else}
        <div class="text-center py-12">
          <p class="text-2xl font-bold">404</p>
          <p class="text-muted-foreground">Page not found</p>
        </div>
      {/if}
    </div>
  </main>
</Sidebar.Provider>

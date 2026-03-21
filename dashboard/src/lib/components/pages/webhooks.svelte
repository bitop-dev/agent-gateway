<script lang="ts">
  import { onMount } from "svelte";
  import { api, type Webhook } from "$lib/api";
  import * as Card from "$lib/components/ui/card/index.js";
  import * as Table from "$lib/components/ui/table/index.js";
  import * as Dialog from "$lib/components/ui/dialog/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Input } from "$lib/components/ui/input/index.js";
  import { Badge } from "$lib/components/ui/badge/index.js";
  import { Skeleton } from "$lib/components/ui/skeleton/index.js";
  import WebhookIcon from "@lucide/svelte/icons/webhook";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import CopyIcon from "@lucide/svelte/icons/copy";

  let webhooks = $state<Webhook[]>([]);
  let loading = $state(true);

  // Create dialog
  let showCreate = $state(false);
  let newName = $state("");
  let newPath = $state("");
  let newProfile = $state("");
  let newTemplate = $state("");
  let newToken = $state("");

  async function refresh() {
    try {
      const resp = await api.getWebhooks();
      webhooks = resp.webhooks || [];
    } catch (e) {
      console.error("Failed to fetch webhooks:", e);
    } finally {
      loading = false;
    }
  }

  onMount(refresh);

  async function create() {
    if (!newName.trim() || !newPath.trim() || !newProfile.trim()) return;
    try {
      await api.createWebhook({
        name: newName.trim(),
        path: newPath.trim(),
        profile: newProfile.trim(),
        taskTemplate: newTemplate.trim(),
        authToken: newToken.trim() || undefined,
        enabled: true,
      });
      newName = newPath = newProfile = newTemplate = newToken = "";
      showCreate = false;
      await refresh();
    } catch (e) {
      console.error("Failed to create webhook:", e);
    }
  }

  async function deleteWebhook(id: string) {
    if (!confirm("Delete this webhook?")) return;
    try {
      await api.deleteWebhook(id);
      await refresh();
    } catch (e) {
      console.error("Failed to delete webhook:", e);
    }
  }

  function copyUrl(path: string) {
    const url = `${window.location.origin}/v1/webhooks/${path}`;
    navigator.clipboard.writeText(url);
  }
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-3xl font-bold">Webhooks</h1>
    <Button size="sm" onclick={() => (showCreate = true)}>
      <PlusIcon class="h-4 w-4 mr-1" />
      New Webhook
    </Button>
  </div>

  <Card.Root>
    <Card.Content class="p-0">
      {#if loading}
        <div class="p-6 space-y-3">
          {#each Array(3) as _}
            <Skeleton class="h-10 w-full" />
          {/each}
        </div>
      {:else if webhooks.length === 0}
        <div class="flex flex-col items-center justify-center py-12 text-center">
          <WebhookIcon class="h-12 w-12 text-muted-foreground mb-4" />
          <p class="text-lg font-medium">No webhooks</p>
          <p class="text-sm text-muted-foreground">
            Create webhooks to trigger tasks from external events
          </p>
        </div>
      {:else}
        <Table.Root>
          <Table.Header>
            <Table.Row>
              <Table.Head>Name</Table.Head>
              <Table.Head>Path</Table.Head>
              <Table.Head>Profile</Table.Head>
              <Table.Head>Template</Table.Head>
              <Table.Head class="w-[80px]">Status</Table.Head>
              <Table.Head class="w-[100px]"></Table.Head>
            </Table.Row>
          </Table.Header>
          <Table.Body>
            {#each webhooks as wh (wh.id)}
              <Table.Row>
                <Table.Cell class="font-medium">{wh.name}</Table.Cell>
                <Table.Cell class="font-mono text-sm"
                  >/{wh.path}</Table.Cell
                >
                <Table.Cell>{wh.profile}</Table.Cell>
                <Table.Cell class="text-sm text-muted-foreground max-w-[200px] truncate">
                  {wh.taskTemplate || "—"}
                </Table.Cell>
                <Table.Cell>
                  <Badge variant={wh.enabled ? "default" : "secondary"}>
                    {wh.enabled ? "active" : "disabled"}
                  </Badge>
                </Table.Cell>
                <Table.Cell>
                  <div class="flex gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      class="h-7 w-7 p-0"
                      onclick={() => copyUrl(wh.path)}
                    >
                      <CopyIcon class="h-3 w-3" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      class="h-7 w-7 p-0 text-destructive"
                      onclick={() => deleteWebhook(wh.id)}
                    >
                      <Trash2Icon class="h-3 w-3" />
                    </Button>
                  </div>
                </Table.Cell>
              </Table.Row>
            {/each}
          </Table.Body>
        </Table.Root>
      {/if}
    </Card.Content>
  </Card.Root>

  <Card.Root>
    <Card.Content class="p-4">
      <p class="text-sm text-muted-foreground">
        <strong>Trigger URL:</strong>
        <code class="ml-1 font-mono text-xs">
          POST {window.location.origin}/v1/webhooks/&#123;path&#125;
        </code>
      </p>
      <p class="text-sm text-muted-foreground mt-1">
        Template variables: <code class="font-mono text-xs">&#123;&#123;key&#125;&#125;</code> and
        <code class="font-mono text-xs">&#123;&#123;nested.key&#125;&#125;</code> from the JSON payload.
      </p>
    </Card.Content>
  </Card.Root>
</div>

<!-- Create dialog -->
<Dialog.Root bind:open={showCreate}>
  <Dialog.Content>
    <Dialog.Header>
      <Dialog.Title>Create Webhook</Dialog.Title>
      <Dialog.Description>
        Create a webhook to trigger tasks from external events
      </Dialog.Description>
    </Dialog.Header>
    <div class="space-y-4 py-4">
      <div>
        <label for="wh-name" class="text-sm font-medium">Name</label>
        <Input id="wh-name" placeholder="grafana-alerts" bind:value={newName} />
      </div>
      <div>
        <label for="wh-path" class="text-sm font-medium">Path</label>
        <Input id="wh-path" placeholder="grafana" bind:value={newPath} />
      </div>
      <div>
        <label for="wh-profile" class="text-sm font-medium">Profile</label>
        <Input
          id="wh-profile"
          placeholder="alert-monitor"
          bind:value={newProfile}
        />
      </div>
      <div>
        <label for="wh-template" class="text-sm font-medium"
          >Task Template</label
        >
        <Input
          id="wh-template"
          placeholder="Alert &#123;&#123;alertname&#125;&#125; on &#123;&#123;host&#125;&#125;"
          bind:value={newTemplate}
        />
      </div>
      <div>
        <label for="wh-token" class="text-sm font-medium"
          >Auth Token (optional)</label
        >
        <Input
          id="wh-token"
          type="password"
          placeholder="secret token"
          bind:value={newToken}
        />
      </div>
    </div>
    <Dialog.Footer>
      <Button variant="outline" onclick={() => (showCreate = false)}
        >Cancel</Button
      >
      <Button
        onclick={create}
        disabled={!newName.trim() || !newPath.trim() || !newProfile.trim()}
      >
        Create
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

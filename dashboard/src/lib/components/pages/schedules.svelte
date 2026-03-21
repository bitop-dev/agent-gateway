<script lang="ts">
  import { onMount } from "svelte";
  import { api, type Schedule } from "$lib/api";
  import * as Card from "$lib/components/ui/card/index.js";
  import * as Table from "$lib/components/ui/table/index.js";
  import * as Dialog from "$lib/components/ui/dialog/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Input } from "$lib/components/ui/input/index.js";
  import { Badge } from "$lib/components/ui/badge/index.js";
  import { Skeleton } from "$lib/components/ui/skeleton/index.js";
  import ClockIcon from "@lucide/svelte/icons/clock";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";

  let schedules = $state<Schedule[]>([]);
  let loading = $state(true);

  // Create dialog
  let showCreate = $state(false);
  let newName = $state("");
  let newCron = $state("");
  let newTimezone = $state("UTC");
  let newProfile = $state("");
  let newTask = $state("");

  async function refresh() {
    try {
      const resp = await api.getSchedules();
      schedules = resp.schedules || [];
    } catch (e) {
      console.error("Failed to fetch schedules:", e);
    } finally {
      loading = false;
    }
  }

  onMount(refresh);

  async function create() {
    if (!newName.trim() || !newCron.trim() || !newProfile.trim() || !newTask.trim())
      return;
    try {
      await api.createSchedule({
        name: newName.trim(),
        cron: newCron.trim(),
        timezone: newTimezone.trim() || "UTC",
        profile: newProfile.trim(),
        task: newTask.trim(),
        enabled: true,
      });
      newName = newCron = newProfile = newTask = "";
      newTimezone = "UTC";
      showCreate = false;
      await refresh();
    } catch (e) {
      console.error("Failed to create schedule:", e);
    }
  }

  async function deleteSchedule(id: string) {
    if (!confirm("Delete this schedule?")) return;
    try {
      await api.deleteSchedule(id);
      await refresh();
    } catch (e) {
      console.error("Failed to delete schedule:", e);
    }
  }
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-3xl font-bold">Schedules</h1>
    <Button size="sm" onclick={() => (showCreate = true)}>
      <PlusIcon class="h-4 w-4 mr-1" />
      New Schedule
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
      {:else if schedules.length === 0}
        <div class="flex flex-col items-center justify-center py-12 text-center">
          <ClockIcon class="h-12 w-12 text-muted-foreground mb-4" />
          <p class="text-lg font-medium">No schedules</p>
          <p class="text-sm text-muted-foreground">
            Create cron schedules to run agents automatically
          </p>
        </div>
      {:else}
        <Table.Root>
          <Table.Header>
            <Table.Row>
              <Table.Head>Name</Table.Head>
              <Table.Head>Cron</Table.Head>
              <Table.Head>Profile</Table.Head>
              <Table.Head>Task</Table.Head>
              <Table.Head>Status</Table.Head>
              <Table.Head>Last Run</Table.Head>
              <Table.Head>Next Run</Table.Head>
              <Table.Head class="w-[60px]"></Table.Head>
            </Table.Row>
          </Table.Header>
          <Table.Body>
            {#each schedules as s (s.id)}
              <Table.Row>
                <Table.Cell class="font-medium">{s.name}</Table.Cell>
                <Table.Cell class="font-mono text-sm">{s.cron}</Table.Cell>
                <Table.Cell>{s.profile}</Table.Cell>
                <Table.Cell class="text-sm text-muted-foreground max-w-[200px] truncate">
                  {s.task}
                </Table.Cell>
                <Table.Cell>
                  <Badge variant={s.enabled ? "default" : "secondary"}>
                    {s.enabled ? "active" : "paused"}
                  </Badge>
                </Table.Cell>
                <Table.Cell class="text-xs text-muted-foreground">
                  {s.lastRun ? new Date(s.lastRun).toLocaleString() : "never"}
                </Table.Cell>
                <Table.Cell class="text-xs text-muted-foreground">
                  {s.nextRun ? new Date(s.nextRun).toLocaleString() : "—"}
                </Table.Cell>
                <Table.Cell>
                  <Button
                    variant="ghost"
                    size="sm"
                    class="h-7 w-7 p-0 text-destructive"
                    onclick={() => deleteSchedule(s.id)}
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

<!-- Create dialog -->
<Dialog.Root bind:open={showCreate}>
  <Dialog.Content>
    <Dialog.Header>
      <Dialog.Title>Create Schedule</Dialog.Title>
      <Dialog.Description>
        Schedule recurring agent tasks with cron expressions
      </Dialog.Description>
    </Dialog.Header>
    <div class="space-y-4 py-4">
      <div>
        <label for="sched-name" class="text-sm font-medium">Name</label>
        <Input
          id="sched-name"
          placeholder="daily-ops-report"
          bind:value={newName}
        />
      </div>
      <div>
        <label for="sched-cron" class="text-sm font-medium">Cron Expression</label>
        <Input
          id="sched-cron"
          placeholder="0 8 * * * (daily at 8am)"
          bind:value={newCron}
        />
      </div>
      <div>
        <label for="sched-tz" class="text-sm font-medium">Timezone</label>
        <Input id="sched-tz" placeholder="UTC" bind:value={newTimezone} />
      </div>
      <div>
        <label for="sched-profile" class="text-sm font-medium">Profile</label>
        <Input
          id="sched-profile"
          placeholder="grafana-alert-summary"
          bind:value={newProfile}
        />
      </div>
      <div>
        <label for="sched-task" class="text-sm font-medium">Task</label>
        <Input
          id="sched-task"
          placeholder="Generate daily ops report"
          bind:value={newTask}
        />
      </div>
    </div>
    <Dialog.Footer>
      <Button variant="outline" onclick={() => (showCreate = false)}
        >Cancel</Button
      >
      <Button
        onclick={create}
        disabled={!newName.trim() ||
          !newCron.trim() ||
          !newProfile.trim() ||
          !newTask.trim()}
      >
        Create
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

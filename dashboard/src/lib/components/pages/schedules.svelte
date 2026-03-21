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
  import PencilIcon from "@lucide/svelte/icons/pencil";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";

  let schedules = $state<Schedule[]>([]);
  let loading = $state(true);

  // Dialog state
  let showDialog = $state(false);
  let editMode = $state(false);
  let editId = $state("");
  let formName = $state("");
  let formCron = $state("");
  let formTimezone = $state("UTC");
  let formProfile = $state("");
  let formTask = $state("");
  let formEnabled = $state(true);
  let saving = $state(false);

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

  function openCreate() {
    editMode = false;
    editId = "";
    formName = "";
    formCron = "";
    formTimezone = "UTC";
    formProfile = "";
    formTask = "";
    formEnabled = true;
    showDialog = true;
  }

  function openEdit(s: Schedule) {
    editMode = true;
    editId = s.id;
    formName = s.name;
    formCron = s.cron;
    formTimezone = s.timezone || "UTC";
    formProfile = s.profile;
    formTask = s.task;
    formEnabled = s.enabled;
    showDialog = true;
  }

  async function save() {
    if (!formName.trim() || !formCron.trim() || !formProfile.trim() || !formTask.trim())
      return;
    saving = true;
    try {
      if (editMode) {
        await api.updateSchedule({
          id: editId,
          name: formName.trim(),
          cron: formCron.trim(),
          timezone: formTimezone.trim() || "UTC",
          profile: formProfile.trim(),
          task: formTask.trim(),
          enabled: formEnabled,
        });
      } else {
        await api.createSchedule({
          name: formName.trim(),
          cron: formCron.trim(),
          timezone: formTimezone.trim() || "UTC",
          profile: formProfile.trim(),
          task: formTask.trim(),
          enabled: true,
        });
      }
      showDialog = false;
      await refresh();
    } catch (e) {
      console.error("Failed to save schedule:", e);
    } finally {
      saving = false;
    }
  }

  async function toggleEnabled(s: Schedule) {
    try {
      await api.updateSchedule({
        id: s.id,
        name: s.name,
        cron: s.cron,
        timezone: s.timezone || "UTC",
        profile: s.profile,
        task: s.task,
        enabled: !s.enabled,
      });
      await refresh();
    } catch (e) {
      console.error("Failed to toggle schedule:", e);
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

  const cronExamples = [
    { label: "Every minute", value: "* * * * *" },
    { label: "Every 5 min", value: "*/5 * * * *" },
    { label: "Every hour", value: "0 * * * *" },
    { label: "Daily 8am", value: "0 8 * * *" },
    { label: "Weekly Mon 9am", value: "0 9 * * 1" },
  ];
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-3xl font-bold">Schedules</h1>
    <Button size="sm" onclick={openCreate}>
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
              <Table.Head class="w-[100px]"></Table.Head>
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
                  <button onclick={() => toggleEnabled(s)}>
                    <Badge
                      variant={s.enabled ? "default" : "secondary"}
                      class="cursor-pointer"
                    >
                      {s.enabled ? "active" : "paused"}
                    </Badge>
                  </button>
                </Table.Cell>
                <Table.Cell class="text-xs text-muted-foreground">
                  {s.lastRun
                    ? new Date(s.lastRun).toLocaleString()
                    : "never"}
                </Table.Cell>
                <Table.Cell class="text-xs text-muted-foreground">
                  {s.nextRun
                    ? new Date(s.nextRun).toLocaleString()
                    : "—"}
                </Table.Cell>
                <Table.Cell>
                  <div class="flex gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      class="h-7 w-7 p-0"
                      onclick={() => openEdit(s)}
                    >
                      <PencilIcon class="h-3 w-3" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      class="h-7 w-7 p-0 text-destructive"
                      onclick={() => deleteSchedule(s.id)}
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
</div>

<!-- Create/Edit dialog -->
<Dialog.Root bind:open={showDialog}>
  <Dialog.Content>
    <Dialog.Header>
      <Dialog.Title>{editMode ? "Edit" : "Create"} Schedule</Dialog.Title>
      <Dialog.Description>
        {editMode
          ? "Update this schedule's configuration"
          : "Schedule recurring agent tasks with cron expressions"}
      </Dialog.Description>
    </Dialog.Header>
    <div class="space-y-4 py-4">
      <div>
        <label for="sched-name" class="text-sm font-medium">Name</label>
        <Input
          id="sched-name"
          placeholder="daily-ops-report"
          bind:value={formName}
        />
      </div>
      <div>
        <label for="sched-cron" class="text-sm font-medium"
          >Cron Expression</label
        >
        <Input
          id="sched-cron"
          placeholder="0 8 * * *"
          bind:value={formCron}
        />
        <div class="flex flex-wrap gap-1 mt-2">
          {#each cronExamples as ex}
            <button
              class="text-[10px] px-2 py-0.5 rounded border border-border hover:bg-accent transition-colors"
              onclick={() => (formCron = ex.value)}
            >
              {ex.label}
            </button>
          {/each}
        </div>
      </div>
      <div>
        <label for="sched-tz" class="text-sm font-medium">Timezone</label>
        <Input id="sched-tz" placeholder="UTC" bind:value={formTimezone} />
      </div>
      <div>
        <label for="sched-profile" class="text-sm font-medium">Profile</label>
        <Input
          id="sched-profile"
          placeholder="researcher"
          bind:value={formProfile}
        />
      </div>
      <div>
        <label for="sched-task" class="text-sm font-medium">Task</label>
        <Input
          id="sched-task"
          placeholder="Generate daily ops report"
          bind:value={formTask}
        />
      </div>
      {#if editMode}
        <div>
          <label class="flex items-center gap-2 text-sm">
            <input type="checkbox" bind:checked={formEnabled} />
            Enabled
          </label>
        </div>
      {/if}
    </div>
    <Dialog.Footer>
      <Button variant="outline" onclick={() => (showDialog = false)}
        >Cancel</Button
      >
      <Button
        onclick={save}
        disabled={!formName.trim() ||
          !formCron.trim() ||
          !formProfile.trim() ||
          !formTask.trim() ||
          saving}
      >
        {saving ? "Saving..." : editMode ? "Save Changes" : "Create"}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

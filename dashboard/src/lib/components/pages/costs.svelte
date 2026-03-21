<script lang="ts">
  import { onMount } from "svelte";
  import { api, type CostSummary, type ModelPricing } from "$lib/api";
  import * as Card from "$lib/components/ui/card/index.js";
  import * as Table from "$lib/components/ui/table/index.js";
  import * as Tabs from "$lib/components/ui/tabs/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Skeleton } from "$lib/components/ui/skeleton/index.js";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import DollarSignIcon from "@lucide/svelte/icons/dollar-sign";

  let costs = $state<CostSummary[]>([]);
  let pricing = $state<ModelPricing[]>([]);
  let loading = $state(true);
  let pricingLoading = $state(true);
  let timeRange = $state("7d");

  function sinceDate(range: string): string {
    const now = new Date();
    switch (range) {
      case "1d":
        return new Date(now.getTime() - 86400000).toISOString().split("T")[0];
      case "7d":
        return new Date(now.getTime() - 7 * 86400000)
          .toISOString()
          .split("T")[0];
      case "30d":
        return new Date(now.getTime() - 30 * 86400000)
          .toISOString()
          .split("T")[0];
      default:
        return "";
    }
  }

  async function refresh() {
    loading = true;
    try {
      const since = sinceDate(timeRange);
      const resp = await api.getCosts(since || undefined) as any;
      // Gateway returns { profiles: [...], totalCost, totalTokens, since }
      costs = resp.costs || resp.profiles || [];
    } catch (e) {
      console.error("Failed to fetch costs:", e);
    } finally {
      loading = false;
    }
  }

  async function loadPricing() {
    pricingLoading = true;
    try {
      const resp = await api.getPricing();
      pricing = resp.pricing || [];
    } catch (e) {
      console.error("Failed to fetch pricing:", e);
    } finally {
      pricingLoading = false;
    }
  }

  onMount(() => {
    refresh();
    loadPricing();
  });

  let totalCost = $derived(costs.reduce((sum, c) => sum + (c.totalCost || c.cost || 0), 0));
  let totalTokensIn = $derived(
    costs.reduce((sum, c) => sum + (c.inputTokens || 0), 0)
  );
  let totalTokensOut = $derived(
    costs.reduce((sum, c) => sum + (c.outputTokens || 0), 0)
  );
  let totalTasks = $derived(
    costs.reduce((sum, c) => sum + (c.totalTasks || c.tasks || 0), 0)
  );

  const ranges = [
    { label: "24h", value: "1d" },
    { label: "7 days", value: "7d" },
    { label: "30 days", value: "30d" },
    { label: "All time", value: "all" },
  ];
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-3xl font-bold">Costs</h1>
    <Button variant="outline" size="sm" onclick={refresh}>
      <RefreshCwIcon class="h-4 w-4" />
    </Button>
  </div>

  <!-- Time range -->
  <div class="flex gap-1">
    {#each ranges as r}
      <Button
        variant={timeRange === r.value ? "default" : "outline"}
        size="sm"
        onclick={() => {
          timeRange = r.value;
          refresh();
        }}
      >
        {r.label}
      </Button>
    {/each}
  </div>

  <!-- Summary cards -->
  <div class="grid gap-4 md:grid-cols-4">
    <Card.Root>
      <Card.Header class="flex flex-row items-center justify-between pb-2">
        <Card.Title class="text-sm font-medium">Total Cost</Card.Title>
        <DollarSignIcon class="h-4 w-4 text-muted-foreground" />
      </Card.Header>
      <Card.Content>
        <div class="text-2xl font-bold">${totalCost.toFixed(4)}</div>
      </Card.Content>
    </Card.Root>
    <Card.Root>
      <Card.Header class="pb-2">
        <Card.Title class="text-sm font-medium">Tasks</Card.Title>
      </Card.Header>
      <Card.Content>
        <div class="text-2xl font-bold">{totalTasks.toLocaleString()}</div>
      </Card.Content>
    </Card.Root>
    <Card.Root>
      <Card.Header class="pb-2">
        <Card.Title class="text-sm font-medium">Tokens In</Card.Title>
      </Card.Header>
      <Card.Content>
        <div class="text-2xl font-bold">
          {totalTokensIn.toLocaleString()}
        </div>
      </Card.Content>
    </Card.Root>
    <Card.Root>
      <Card.Header class="pb-2">
        <Card.Title class="text-sm font-medium">Tokens Out</Card.Title>
      </Card.Header>
      <Card.Content>
        <div class="text-2xl font-bold">
          {totalTokensOut.toLocaleString()}
        </div>
      </Card.Content>
    </Card.Root>
  </div>

  <Tabs.Root value="by-profile">
    <Tabs.List>
      <Tabs.Trigger value="by-profile">By Profile</Tabs.Trigger>
      <Tabs.Trigger value="pricing">Model Pricing</Tabs.Trigger>
    </Tabs.List>

    <Tabs.Content value="by-profile">
      <Card.Root>
        <Card.Content class="p-0">
          {#if loading}
            <div class="p-6 space-y-3">
              {#each Array(3) as _}
                <Skeleton class="h-10 w-full" />
              {/each}
            </div>
          {:else if costs.length === 0}
            <div class="p-8 text-center text-muted-foreground">
              No cost data for this period
            </div>
          {:else}
            <Table.Root>
              <Table.Header>
                <Table.Row>
                  <Table.Head>Profile</Table.Head>
                  <Table.Head class="text-right">Tasks</Table.Head>
                  <Table.Head class="text-right">Tokens In</Table.Head>
                  <Table.Head class="text-right">Tokens Out</Table.Head>
                  <Table.Head class="text-right">Cost</Table.Head>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {#each costs as c (c.profile)}
                  <Table.Row>
                    <Table.Cell class="font-medium">{c.profile}</Table.Cell>
                    <Table.Cell class="text-right">{c.totalTasks || c.tasks}</Table.Cell>
                    <Table.Cell class="text-right"
                      >{(c.inputTokens || 0).toLocaleString()}</Table.Cell
                    >
                    <Table.Cell class="text-right"
                      >{(c.outputTokens || 0).toLocaleString()}</Table.Cell
                    >
                    <Table.Cell class="text-right font-medium"
                      >${(c.totalCost || c.cost || 0).toFixed(6)}</Table.Cell
                    >
                  </Table.Row>
                {/each}
              </Table.Body>
            </Table.Root>
          {/if}
        </Card.Content>
      </Card.Root>
    </Tabs.Content>

    <Tabs.Content value="pricing">
      <Card.Root>
        <Card.Content class="p-0">
          {#if pricingLoading}
            <div class="p-6 space-y-3">
              {#each Array(5) as _}
                <Skeleton class="h-8 w-full" />
              {/each}
            </div>
          {:else if pricing.length === 0}
            <div class="p-8 text-center text-muted-foreground">
              No pricing data loaded
            </div>
          {:else}
            <Table.Root>
              <Table.Header>
                <Table.Row>
                  <Table.Head>Model</Table.Head>
                  <Table.Head class="text-right">Input $/M</Table.Head>
                  <Table.Head class="text-right">Output $/M</Table.Head>
                  <Table.Head>Source</Table.Head>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {#each pricing.slice(0, 50) as p (p.model)}
                  <Table.Row>
                    <Table.Cell class="font-mono text-sm">{p.model}</Table.Cell>
                    <Table.Cell class="text-right"
                      >${p.inputPerMillion.toFixed(2)}</Table.Cell
                    >
                    <Table.Cell class="text-right"
                      >${p.outputPerMillion.toFixed(2)}</Table.Cell
                    >
                    <Table.Cell class="text-xs text-muted-foreground"
                      >{p.source}</Table.Cell
                    >
                  </Table.Row>
                {/each}
              </Table.Body>
            </Table.Root>
            {#if pricing.length > 50}
              <div class="p-3 text-center text-sm text-muted-foreground">
                Showing 50 of {pricing.length} models
              </div>
            {/if}
          {/if}
        </Card.Content>
      </Card.Root>
    </Tabs.Content>
  </Tabs.Root>
</div>

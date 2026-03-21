<script lang="ts">
  import LayoutDashboardIcon from "@lucide/svelte/icons/layout-dashboard";
  import ListTodoIcon from "@lucide/svelte/icons/list-todo";
  import ServerIcon from "@lucide/svelte/icons/server";
  import BotIcon from "@lucide/svelte/icons/bot";
  import DollarSignIcon from "@lucide/svelte/icons/dollar-sign";
  import BrainIcon from "@lucide/svelte/icons/brain";
  import WebhookIcon from "@lucide/svelte/icons/webhook";
  import ClockIcon from "@lucide/svelte/icons/clock";
  import PuzzleIcon from "@lucide/svelte/icons/puzzle";
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import { currentPath, navigate } from "$lib/router";

  const navItems = [
    { title: "Overview", href: "/", icon: LayoutDashboardIcon },
    { title: "Tasks", href: "/tasks", icon: ListTodoIcon },
    { title: "Workers", href: "/workers", icon: ServerIcon },
    { title: "Agents", href: "/agents", icon: BotIcon },
    { title: "Plugins", href: "/plugins", icon: PuzzleIcon },
    { title: "Costs", href: "/costs", icon: DollarSignIcon },
    { title: "Memory", href: "/memory", icon: BrainIcon },
    { title: "Webhooks", href: "/webhooks", icon: WebhookIcon },
    { title: "Schedules", href: "/schedules", icon: ClockIcon },
  ];

  function isActive(href: string, path: string): boolean {
    if (href === "/") return path === "/";
    return path.startsWith(href);
  }
</script>

<Sidebar.Root>
  <Sidebar.Header>
    <div class="flex items-center gap-2 px-2 py-3">
      <BotIcon class="h-6 w-6 text-primary" />
      <span class="text-lg font-semibold">Agent Platform</span>
    </div>
  </Sidebar.Header>
  <Sidebar.Content>
    <Sidebar.Group>
      <Sidebar.GroupContent>
        <Sidebar.Menu class="gap-1">
          {#each navItems as item (item.title)}
            <Sidebar.MenuItem>
              <Sidebar.MenuButton
                size="lg"
                isActive={isActive(item.href, $currentPath)}
              >
                {#snippet child({ props })}
                  <a
                    href="#{item.href}"
                    {...props}
                    class="{props.class} py-3"
                    onclick={(e: MouseEvent) => {
                      e.preventDefault();
                      navigate(item.href);
                    }}
                  >
                    <item.icon />
                    <span>{item.title}</span>
                  </a>
                {/snippet}
              </Sidebar.MenuButton>
            </Sidebar.MenuItem>
          {/each}
        </Sidebar.Menu>
      </Sidebar.GroupContent>
    </Sidebar.Group>
  </Sidebar.Content>
  <Sidebar.Footer>
    <div class="px-2 py-3 text-xs text-muted-foreground">
      Agent Platform v0.5.2
    </div>
  </Sidebar.Footer>
</Sidebar.Root>

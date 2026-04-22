export interface DashboardCommand {
  label: string
  command: string
}

export const dashboardCommands: DashboardCommand[] = [
  {
    label: 'Start desktop development',
    command: 'wails dev',
  },
  {
    label: 'Install frontend dependencies with Deno',
    command: 'cd frontend && deno install',
  },
  {
    label: 'Build the Vue frontend',
    command: 'cd frontend && deno task build',
  },
  {
    label: 'Compile the desktop app',
    command: 'wails build',
  },
]

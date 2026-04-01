export interface PlotPreset {
  plot_id: string
  plot_name: string
}

export interface OrchardPreset {
  orchard_id: string
  orchard_name: string
  plots: PlotPreset[]
}

export const ORCHARD_PRESETS: OrchardPreset[] = [
  {
    orchard_id: 'orchard-demo-01',
    orchard_name: '荔枝示范园',
    plots: [
      { plot_id: 'plot-a01', plot_name: 'A1 区' },
      { plot_id: 'plot-a02', plot_name: 'A2 区' }
    ]
  },
  {
    orchard_id: 'orchard-east-02',
    orchard_name: '东麓果园',
    plots: [
      { plot_id: 'plot-e01', plot_name: '东坡 1 号地块' },
      { plot_id: 'plot-e02', plot_name: '东坡 2 号地块' }
    ]
  },
  {
    orchard_id: 'orchard-north-03',
    orchard_name: '北山合作社果园',
    plots: [
      { plot_id: 'plot-n01', plot_name: '北山上层区' },
      { plot_id: 'plot-n02', plot_name: '北山下层区' }
    ]
  }
]

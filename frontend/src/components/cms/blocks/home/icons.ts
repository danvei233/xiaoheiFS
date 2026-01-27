import { defineComponent, h } from 'vue'

export const ThunderIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('path', {
      d: 'M13 2L3 14H12L11 22L21 10H12L13 2Z',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round'
    })
  ])
})

export const ShieldIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('path', {
      d: 'M12 22S3 18 3 6V3L12 1L21 3V6C21 18 12 22 12 22Z',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round'
    })
  ])
})

export const GlobeIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('circle', { cx: '12', cy: '12', r: '10', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M2 12H22', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', {
      d: 'M12 2C14.5013 4.73835 15.9228 8.29203 16 12C15.9228 15.708 14.5013 19.2616 12 22C9.49872 19.2616 8.07725 15.708 8 12C8.07725 8.29203 9.49872 4.73835 12 2Z',
      stroke: 'currentColor',
      'stroke-width': 2
    })
  ])
})

export const ServerIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('rect', { x: '2', y: '2', width: '20', height: '8', rx: '2', stroke: 'currentColor', 'stroke-width': 2 }),
    h('rect', { x: '2', y: '14', width: '20', height: '8', rx: '2', stroke: 'currentColor', 'stroke-width': 2 }),
    h('line', { x1: '6', y1: '6', x2: '6.01', y2: '6', stroke: 'currentColor', 'stroke-width': 3, 'stroke-linecap': 'round' }),
    h('line', { x1: '6', y1: '18', x2: '6.01', y2: '18', stroke: 'currentColor', 'stroke-width': 3, 'stroke-linecap': 'round' })
  ])
})

export const DatabaseIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('ellipse', { cx: '12', cy: '5', rx: '9', ry: '3', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M21 12C21 13.66 16.97 15 12 15C7.03 15 3 13.66 3 12', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M3 5V19C3 20.66 7.03 22 12 22C16.97 22 21 20.66 21 19V5', stroke: 'currentColor', 'stroke-width': 2 })
  ])
})

export const SettingsIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('circle', { cx: '12', cy: '12', r: '3', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', {
      d: 'M19.4 15A1.65 1.65 0 0 0 21 13.35V10.65A1.65 1.65 0 0 0 19.4 9L18.34 8.47A1.65 1.65 0 0 1 17.65 6.87L18.12 5.16A1.65 1.65 0 0 0 16.88 3.12L15.17 3.59A1.65 1.65 0 0 1 13.57 2.9L13.04 1.84A1.65 1.65 0 0 0 11.35 0.25H8.65A1.65 1.65 0 0 0 6.96 1.84L6.43 2.9A1.65 1.65 0 0 1 4.83 3.59L3.12 3.12A1.65 1.65 0 0 0 1.88 5.16L2.35 6.87A1.65 1.65 0 0 1 1.66 8.47L0.6 9A1.65 1.65 0 0 0 -1 10.65V13.35A1.65 1.65 0 0 0 0.6 15L1.66 15.53A1.65 1.65 0 0 1 2.35 17.13L1.88 18.84A1.65 1.65 0 0 0 3.12 20.88L4.83 20.41A1.65 1.65 0 0 1 6.43 21.1L6.96 22.16A1.65 1.65 0 0 0 8.65 23.75H11.35A1.65 1.65 0 0 0 13.04 22.16L13.57 21.1A1.65 1.65 0 0 1 15.17 20.41L16.88 20.88A1.65 1.65 0 0 0 18.12 18.84L17.65 17.13A1.65 1.65 0 0 1 18.34 15.53L19.4 15Z',
      stroke: 'currentColor',
      'stroke-width': 2,
      transform: 'translate(2 0)'
    })
  ])
})

export const CloudIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('path', {
      d: 'M18 10H17.5C17.1 6 13.5 3 9.5 3C5.5 3 2 6 2 10C2 10.5 2.1 11 2.2 11.5C1 12 0 13.2 0 14.5C0 16.5 1.5 18 3.5 18H18C20.8 18 23 15.8 23 13C23 10.5 21.2 8.5 18.8 8.1C18.6 8 18.3 8 18 8V10Z',
      stroke: 'currentColor',
      'stroke-width': 2,
      transform: 'translate(1 2)'
    })
  ])
})

export const CubeIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('path', { d: 'M12 2L2 7L12 12L22 7L12 2Z', stroke: 'currentColor', 'stroke-width': 2, 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
    h('path', { d: 'M2 17L12 22L22 17', stroke: 'currentColor', 'stroke-width': 2, 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
    h('path', { d: 'M2 12L12 17L22 12', stroke: 'currentColor', 'stroke-width': 2, 'stroke-linecap': 'round', 'stroke-linejoin': 'round' })
  ])
})

export const CodeIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('polyline', { points: '16 18 22 12 16 6', stroke: 'currentColor', 'stroke-width': 2, 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
    h('polyline', { points: '8 6 2 12 8 18', stroke: 'currentColor', 'stroke-width': 2, 'stroke-linecap': 'round', 'stroke-linejoin': 'round' })
  ])
})

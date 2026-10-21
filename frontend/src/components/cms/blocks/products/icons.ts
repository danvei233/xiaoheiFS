import { defineComponent, h } from 'vue'

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

export const RocketIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('path', {
      d: 'M4.5 16.5C4.5 16.5 4 14 4 14',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round'
    }),
    h('path', {
      d: 'M7.5 19.5C7.5 19.5 7 17 7 17',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round'
    }),
    h('path', {
      d: 'M16.5 10C16.5 10 19 12 19 14C19 17 16.5 19 16.5 19',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round'
    }),
    h('path', {
      d: 'M15.5 4L13 10L7 10L4.5 4',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round'
    }),
    h('path', {
      d: 'M13 10V19C13 20.66 11.66 22 10 22C8.34 22 7 20.66 7 19V10',
      stroke: 'currentColor',
      'stroke-width': 2
    }),
    h('path', {
      d: 'M10 2V4',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round'
    }),
    h('path', {
      d: 'M16 6L13 10',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round'
    }),
    h('path', {
      d: 'M4 6L7 10',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round'
    })
  ])
})

export const BoltIcon = defineComponent({
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

export const BuildingIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('rect', { x: '4', y: '2', width: '16', height: '20', rx: '2', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M9 2V22', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M15 2V22', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M4 12H20', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M4 7H9', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M15 7H20', stroke: 'currentColor', 'stroke-width': 2 })
  ])
})

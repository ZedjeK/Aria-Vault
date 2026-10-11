<script>
  // Logo d'Aria Vault : le A glitché de la famille Aria, avec la mallette blindée orange.
  // Chaque instance a ses propres identifiants SVG (découpes, filtres).
  let { size = 36, dim = false, label = '' } = $props()

  const id = `logo${Math.random().toString(36).slice(2, 10)}`
  const ref = (name) => `${id}-${name}`
</script>

<svg
  width={size}
  height={size}
  viewBox="-8 -8 176 176"
  class:dim
  role={label ? 'img' : undefined}
  aria-label={label || undefined}
  aria-hidden={label ? undefined : 'true'}
>
  <defs>
    <pattern id={ref('s')} width="4" height="4" patternUnits="userSpaceOnUse">
      <rect width="4" height="1.5" fill="#7DF9FF" opacity="0.07" />
    </pattern>
    <polygon id={ref('a')} points="30,130 70,30 90,30 130,130 110,130 80,54 50,130" />
    <clipPath id={ref('bl')}><rect x="0" y="72" width="80" height="90" /></clipPath>
    <clipPath id={ref('br')}><rect x="80" y="72" width="80" height="90" /></clipPath>
    <clipPath id={ref('t')}><rect x="0" y="0" width="160" height="72" /></clipPath>
    <filter id={ref('g')} x="-40%" y="-40%" width="180%" height="180%">
      <feGaussianBlur stdDeviation="3.2" result="b" />
      <feMerge><feMergeNode in="b" /><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge>
    </filter>
    <filter id={ref('g2')} x="-20%" y="-20%" width="140%" height="140%">
      <feGaussianBlur stdDeviation="2" result="b" />
      <feMerge><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge>
    </filter>
  </defs>

  <polygon points="24,0 136,0 160,24 160,136 136,160 24,160 0,136 0,24" fill="#030812" />
  <polygon points="24,0 136,0 160,24 160,136 136,160 24,160 0,136 0,24" fill="url(#{ref('s')})" />
  <g class="a-part">
    <polygon
      points="26,7 134,7 153,26 153,134 134,153 26,153 7,134 7,26"
      fill="none"
      stroke="#4D7CFF"
      stroke-width="1.6"
      opacity="0.6"
      filter="url(#{ref('g2')})"
    />
    <g filter="url(#{ref('g')})" stroke-linejoin="round">
      <g clip-path="url(#{ref('bl')})" fill="none" stroke="#7DF9FF" stroke-width="4"><use href="#{ref('a')}" /></g>
      <g clip-path="url(#{ref('br')})" fill="none" stroke="#5C8BFF" stroke-width="4"><use href="#{ref('a')}" /></g>
      <g transform="translate(7,0)">
        <g clip-path="url(#{ref('t')})" fill="none" stroke="#FFFFFF" stroke-width="4"><use href="#{ref('a')}" /></g>
      </g>
    </g>
  </g>
  <!-- La mallette reste orange même quand le logo est éteint : Aria veille. -->
  <g transform="translate(80 104) scale(1.18) translate(-80 -104)" filter="url(#{ref('g2')})">
    <path d="M74,96 v-4.5 a2 2 0 0 1 2,-2 h8 a2 2 0 0 1 2,2 v4.5" fill="none" stroke="#FF8A1F" stroke-width="2.4" />
    <rect x="65" y="96" width="30" height="22" rx="3" fill="#FF8A1F" />
    <g fill="#030812" opacity="0.6">
      <rect x="65" y="104.5" width="30" height="1.8" />
      <rect x="70" y="99.5" width="4" height="3" rx="0.8" />
      <rect x="86" y="99.5" width="4" height="3" rx="0.8" />
    </g>
  </g>
</svg>

<style>
  svg {
    display: block;
    flex: none;
  }
  .a-part {
    transition: filter 0.3s, opacity 0.3s;
  }
  /* Éteint : le A s'efface, la mallette orange reste allumée */
  .dim .a-part {
    filter: saturate(0.3);
    opacity: 0.5;
  }
</style>

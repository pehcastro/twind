# Layout

Twind lays out with flexbox and grid, the same model as the browser, in whole terminal cells.

## Sizes are cells

One spacing step is one cell. `w-10` is ten columns wide, `h-1` is one row tall, `px-2` pads two columns on each side and `gap-1` leaves one cell between children. Fractions and percentages divide the parent's width and round to whole cells, so `w-1/2` of a 41-cell box is 20 or 21 cells, never half a cell.

Breakpoints are cells too: `sm:` applies from 80 columns, `md:` from 100, `lg:` from 140 and `xl:` from 180. A resize lays the tree out again with the new width.

## Flexbox

`flex flex-row` and `flex flex-col` set the direction. `grow`, `shrink-0`, `basis-*`, `justify-*`, `items-*`, `self-*`, `flex-wrap` and `gap-*` work as they do on the web.

<Preview name="layout-flex" />

## Grid

`grid grid-cols-3` makes three equal tracks. `col-span-*`, `row-span-*`, `grid-rows-*` and `gap-*` place the children.

<Preview name="layout-grid" />

## Position and overflow

`relative`, `absolute` and `fixed` with `top-*`, `right-*`, `bottom-*`, `left-*` and `inset-*` take an element out of the flow. `z-*` decides what draws on top. `overflow-hidden` clips a child at the border; `overflow-auto` and `overflow-y-auto` make the box scroll with the wheel, the arrows and PageUp and PageDown.

<Preview name="layout-position" />

## Borders and surfaces

`border` reserves a ring of one cell around the box, like a 1px border in the browser. Where the terminal can show images, backgrounds, borders, rounded corners and shadows are drawn as pixels under the text, which stays real text in your font. Elsewhere they are drawn in cells, over the same boxes, so nothing moves between terminals: the border is a box-drawing line `─ │` on that ring with `╭ ╮ ╰ ╯` where a corner is rounded, and the page shows behind it; a filled box with a solid border on every side keeps its background inside the line, any other box fills its whole box; a focus ring or a shadow is a half block `▄ ▌ ▀ ▐` or a shaded cell; and a one-row `rounded-full` pill is its background across the whole box. Every one of these characters is in Consolas, Cascadia Mono and JetBrains Mono. With 16 colours, a surface, border or text that would land on its background's palette colour is moved to the next one, and a symbol the console font lacks, such as `⌄` or `◐`, is drawn as one it has, such as `▾` or `●`.

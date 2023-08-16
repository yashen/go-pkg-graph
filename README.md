生成package之间的关系依赖图

代码重构的时候我想画出图来以方便思考，先找一下有没有现成的

先是发现了[godag](https://github.com/legendtkl/godag)

感觉是我想要的，但是代码已经2018年的，当前并不好用


然后又找到了[go-callvis](https://github.com/ondrajz/go-callvis)

然而使用效果并不太理想，功能很强大，但和我想要的功能侧重点不同，所以就参考它们自己写一个

## 安装
`
go install github.com/yashen/go-pkg-graph@latest
`

## 使用

使用前先安装[Graphviz](https://www.graphviz.org/download/),确保dot命令可用

```
go-pkg-graph --help   

Usage: go-pkg-graph [OPTIONS]

gen package depend graph

Options:
  -o, --output   output svg file (default "graph.svg")
  -d, --depth    package depth (default 3)
  -m, --module   module path (default ".")

```
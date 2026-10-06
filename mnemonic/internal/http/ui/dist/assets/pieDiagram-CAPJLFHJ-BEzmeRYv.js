import{p as et}from"./chunk-JWPE2WC7-DDB-lXb3.js";import{g as at,s as rt,a as it,b as nt,q as ot,p as st,_ as l,l as E,c as lt,O as ct,R as dt,S as gt,d as pt,y as ht,P as ft}from"./mermaid.core-DHD3Y4C_.js";import{p as ut}from"./cynefin-EF2NZ3EQ-BeH8bUEV.js";import{d as U,o as mt,a as vt}from"./vendor-d3-BPSgmlxF.js";import"./index-Dy40kayC.js";import"./vendor-react-BhEX1gNo.js";import"./MarkdownView-CMkUWkI3.js";var St=ft.pie,R={sections:new Map,showData:!1},T=R.sections,L=R.showData,xt=structuredClone(St),wt=l(()=>structuredClone(xt),"getConfig"),Ct=l(()=>{T=new Map,L=R.showData,ht()},"clear"),$t=l(({label:t,value:a})=>{if(a<0)throw new Error(`"${t}" has invalid value: ${a}. Negative values are not allowed in pie charts. All slice values must be >= 0.`);T.has(t)||(T.set(t,a),E.debug(`added new section: ${t}, with value: ${a}`))},"addSection"),yt=l(()=>T,"getSections"),Dt=l(t=>{L=t},"setShowData"),Tt=l(()=>L,"getShowData"),q={getConfig:wt,clear:Ct,setDiagramTitle:st,getDiagramTitle:ot,setAccTitle:nt,getAccTitle:it,setAccDescription:rt,getAccDescription:at,addSection:$t,getSections:yt,setShowData:Dt,getShowData:Tt},bt=l((t,a)=>{et(t,a),a.setShowData(t.showData),t.sections.map(a.addSection)},"populateDb"),At={parse:l(async t=>{const a=await ut("pie",t);E.debug(a),bt(a,q)},"parse")},_t=l(t=>`
  .pieCircle{
    stroke: ${t.pieStrokeColor};
    stroke-width : ${t.pieStrokeWidth};
    opacity : ${t.pieOpacity};
  }
  .pieCircle.highlighted{
    scale: 1.05;
    opacity: 1;
  }
  .pieCircle.highlightedOnHover:hover{
    transition-duration: 250ms;
    scale: 1.05;
    opacity: 1;
  }
  .pieOuterCircle{
    stroke: ${t.pieOuterStrokeColor};
    stroke-width: ${t.pieOuterStrokeWidth};
    fill: none;
  }
  .pieTitleText {
    text-anchor: middle;
    font-size: ${t.pieTitleTextSize};
    fill: ${t.pieTitleTextColor};
    font-family: ${t.fontFamily};
  }
  .slice {
    font-family: ${t.fontFamily};
    fill: ${t.pieSectionTextColor};
    font-size:${t.pieSectionTextSize};
    // fill: white;
  }
  .legend text {
    fill: ${t.pieLegendTextColor};
    font-family: ${t.fontFamily};
    font-size: ${t.pieLegendTextSize};
  }
`,"getStyles"),kt=_t,zt=l(t=>{const a=[...t.values()].reduce((o,m)=>o+m,0),O=[...t.entries()].map(([o,m])=>({label:o,value:m})).filter(o=>o.value/a*100>=1);return vt().value(o=>o.value).sort(null)(O)},"createPieArcs"),Et=l((t,a,O,P)=>{var I;E.debug(`rendering pie chart
`+t);const o=P.db,m=lt(),h=ct(o.getConfig(),m.pie),W=40,i=18,c=4,C=450,S=C,b=dt(a),$=b.append("g");$.attr("transform","translate("+S/2+","+C/2+")");const{themeVariables:n}=m;let[F]=gt(n.pieOuterStrokeWidth);F??(F=2);const V=h.legendPosition,H=h.textPosition,X=h.donutHole>0&&h.donutHole<=.9?h.donutHole:0,f=Math.min(S,C)/2-W,Z=U().innerRadius(X*f).outerRadius(f),j=U().innerRadius(f*H).outerRadius(f*H),x=$.append("g");x.append("circle").attr("cx",0).attr("cy",0).attr("r",f+F/2).attr("class","pieOuterCircle");const y=o.getSections(),J=zt(y),K=[n.pie1,n.pie2,n.pie3,n.pie4,n.pie5,n.pie6,n.pie7,n.pie8,n.pie9,n.pie10,n.pie11,n.pie12];let A=0;y.forEach(e=>{A+=e});const M=J.filter(e=>(e.data.value/A*100).toFixed(0)!=="0"),_=mt(K).domain([...y.keys()]);x.selectAll("mySlices").data(M).enter().append("path").attr("d",Z).attr("fill",e=>_(e.data.label)).attr("class",e=>{let r="pieCircle";return h.highlightSlice==="hover"?r+=" highlightedOnHover":h.highlightSlice===e.data.label&&(r+=" highlighted"),r}),x.selectAll("mySlices").data(M).enter().append("text").text(e=>(e.data.value/A*100).toFixed(0)+"%").attr("transform",e=>"translate("+j.centroid(e)+")").style("text-anchor","middle").attr("class","slice");const Q=$.append("text").text(o.getDiagramTitle()).attr("x",0).attr("y",-400/2).attr("class","pieTitleText"),w=[...y.entries()].map(([e,r])=>({label:e,value:r})),u=$.selectAll(".legend").data(w).enter().append("g").attr("class","legend");u.append("rect").attr("width",i).attr("height",i).style("fill",e=>_(e.label)).style("stroke",e=>_(e.label)),u.append("text").attr("x",i+c).attr("y",i-c).text(e=>o.getShowData()?`${e.label} [${e.value}]`:e.label);const v=Math.max(...u.selectAll("text").nodes().map(e=>(e==null?void 0:e.getBoundingClientRect().width)??0));let D=C,k=S+W;const s=i+c,z=w.length*s;switch(V){case"center":u.attr("transform",(e,r)=>{const d=s*w.length/2,g=-v/2-(i+c),p=r*s-d;return"translate("+g+","+p+")"});break;case"top":D+=z,u.attr("transform",(e,r)=>{const d=f,g=-v/2-(i+c),p=r*s-d;return`translate(${g}, ${p})`}),x.attr("transform",()=>`translate(0, ${z+s})`);break;case"bottom":D+=z,u.attr("transform",(e,r)=>{const d=-f-s,g=-v/2-(i+c),p=r*s-d;return"translate("+g+","+p+")"});break;case"left":k+=i+c+v,u.attr("transform",(e,r)=>{const d=s*w.length/2,g=-f-(i+c),p=r*s-d;return"translate("+g+","+p+")"}),x.attr("transform",()=>`translate(${v+i+c}, 0)`);break;case"right":default:k+=i+c+v,u.attr("transform",(e,r)=>{const d=s*w.length/2,g=12*i,p=r*s-d;return"translate("+g+","+p+")"});break}const G=((I=Q.node())==null?void 0:I.getBoundingClientRect().width)??0,Y=S/2-G/2,tt=S/2+G/2,B=Math.min(0,Y),N=Math.max(k,tt)-B;b.attr("viewBox",`${B} 0 ${N} ${D}`),pt(b,D,N,h.useMaxWidth)},"draw"),Rt={draw:Et},Bt={parser:At,db:q,renderer:Rt,styles:kt};export{Bt as diagram};

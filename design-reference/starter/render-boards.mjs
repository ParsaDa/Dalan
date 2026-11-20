import { chromium } from '/home/claude/dalan/frontend/node_modules/playwright-core/index.mjs';
const names = ["Main","Phone-Dashboard","Motion","Loading","Brand","Login","Phone-Login","Wizard","Wizard-Phone","Error","Sub","Sub-FA","Sub-Light","Sub-Laptop","Inbounds","Clients","Groups","Hosts","Activity","Tunnels","Outbounds","Nodes","Node-Edit","TorPIA","XrayConfig","Logs","Settings","Developer","States","Nav-Modes","Dashboard-Light","Clients-Light","Dashboard-FA","Phone-Dashboard-FA","Phone-Clients","Phone-Tunnels","Phone-Settings","Phone-Search"];
const b = await chromium.launch({ executablePath: '/usr/bin/google-chrome', args: ['--no-sandbox'] });
const pg = await (await b.newContext({ viewport: { width:1600, height:1000 } })).newPage();
await pg.goto('file:///tmp/dl-ref/design.html', { waitUntil: 'load' });
await pg.waitForTimeout(5000);
const n = await pg.evaluate(()=>document.querySelectorAll('iframe').length);
for (let i=0;i<n;i++){
  const h = (await pg.$$('iframe'))[i];
  await h.scrollIntoViewIfNeeded(); await pg.waitForTimeout(1200);
  const box = await h.boundingBox();
  try { await h.screenshot({ path: `/tmp/dl-ref/ref/${names[i]}.png`, timeout: 20000 }); console.log(names[i], Math.round(box.width), Math.round(box.height)); }
  catch(e){ console.log('fail', names[i], e.message.slice(0,80)); }
}
await b.close();

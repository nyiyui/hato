const platformDisplay = document.getElementById("platform-display");
const srcUrl = document.getElementById("src-url");
const srcUrlSubmit = document.getElementById("src-url-submit");
let src = null;
srcUrlSubmit.addEventListener("click", updateSource);

const initialReconnectTimeout = 500;
const maxReconnectTimeout = 32000;
let reconnectTimeout = initialReconnectTimeout;

const statusElem = document.getElementById("status");

function updateSource() {
  if (src) src.close();
  statusElem.textContent = "connecting…";
  src = new EventSource(srcUrl.value);
  src.addEventListener("updateAlloc", newAlloc);
  src.addEventListener("open", (e) => {
    reconnectTimeout = initialReconnectTimeout
    statusElem.textContent = "connected";
  });
  src.addEventListener("error", (e) => {
    reconnectTimeout *= 2;
    if (reconnectTimeout > maxReconnectTimeout) reconnectTimeout = maxReconnectTimeout;
    console.log(`connnecting failed; retry in ${reconnectTimeout}ms...`);
    setTimeout(updateSource, reconnectTimeout);
    statusElem.textContent = `connection: retry in ${reconnectTimeout/1000}s`;
  });
  console.log(`new source URL: ${srcUrl.value}`);
}

let allocs = [
  //{type: "普通", index: "0G39", time: Date.now() + 60000, track: "1", dir: "永瀬"},
  //{type: "普通", index: "1G42", time: Date.now() + 120000, track: "1", dir: "永瀬"},
];

function newAlloc(e) {
  allocs = JSON.parse(e.data)
  console.log("newAlloc: ", allocs);
}

function updateAllocs() {
  const now = Date.now();
  platformDisplay.setAttribute('allocs', JSON.stringify(allocs.map((allocMutable) => {
    let alloc = structuredClone(allocMutable);
    const seconds = Math.floor((allocMutable.time - now)/1000);
    if (seconds < 0) seconds = 0;
    alloc.time = `約${Seconds}秒後`;
    return alloc;
  })));
}

setInterval(updateAllocs, 1000);

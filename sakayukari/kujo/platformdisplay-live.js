const platformDisplay = document.getElementById("platform-display");
const srcUrl = document.getElementById("src-url");
const stopID = document.getElementById("stop-id");
const srcUrlSubmit = document.getElementById("src-url-submit");
let src = null;
srcUrlSubmit.addEventListener("click", updateSource);

function updateSource() {
  if (src) src.close();
  src = new EventSource(srcUrl.value);
  src.addEventListener("newAlloc", newAlloc);
  console.log(`new source URL: ${srcUrl.value}`);
}

let allocs = [
  {type: "普通", index: "0G39", time: Date.now() + 60000, track: "1", dir: "永瀬"},
  {type: "普通", index: "1G42", time: Date.now() + 120000, track: "1", dir: "永瀬"},
];

function newAlloc(e) {
  allocs = JSON.parse(e.data)
  console.log("newAlloc: ", allocs);
}

function updateAllocs() {
  const now = Date.now();
  platformDisplay.setAttribute('allocs', JSON.stringify(allocs.map((allocMutable) => {
    let alloc = structuredClone(allocMutable);
    alloc.time = `約${Math.floor((allocMutable.time - now)/1000)}秒後`;
    return alloc;
  })));
}

setInterval(updateAllocs, 1000);

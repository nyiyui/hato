const styleText = `
.wrapper {
  margin: 0;
  padding: 0;
  color: #fff;
  background: #000;
}

.name {
  color: #eee;
}

.place {
  margin: 0;
  color: #fff;
  background: #416fc4;
  display: flex;
}

.num {
  font-size: 28px;
  margin: 4px;
  margin-right: 12px;
}

.name {
  margin: 4px;
  margin-right: auto;
  font-size: 20px;
}

.clock-label {
  font-size: 8px;
}

.clock {
  background: #000;
  font-family: monospace; /* we want the hyphens to be monospace too */
  padding: 4px;
  height: 100%;
  align-self: end;
}

.allocs {
  width: 100%;
}

.allocs th {
  font-size: 12px;
  font-weight: normal;
}

.allocs td {
  font-size: 20px;
}

.allocs th, .allocs td {
  text-align: center;
  vertical-align: 50%;
}

.allocs td.type[data-value="各停"] { color: #0f0; }

.allocs td.type[data-value="急行"] { color: #ff0; }

.allocs td.type[data-value="回送"] { color: #ddd; }

.allocs td.time {
  color: #0f0;
}

.scroll {
  color: #fff;
  white-space: nowrap;
  overflow: hidden;
  display: inline-block;
  padding-left: 100%;

  animation: scroll-anim 20s linear infinite;
}

.scroll strong {
  color: yellow;
  font-weight: normal;
}

@keyframes scroll-anim {
  from {
    transform: translateX(0);
  }
  to {
    transform: translateX(-100%);
  }
`;

const tenji = '点字タイルは目の不自由な方たちの、<strong>大切な道標</strong>です。<strong>タイルの上に立ち止まったり、バッグなどを置いたりしないよう</strong>、ご協力をお願いします。';

const scrolls = [
  '点字タイルは目の不自由な方たちの、<strong>大切な道標</strong>です。<strong>タイルの上に立ち止まったり、バッグなどを置いたりしないよう</strong>、ご協力をお願いします。',
  '模型列車が<strong>長時間停車</strong>しているときは、発表者にお知らせください。',
  '<strong>冷房が苦手なお客様</strong>にも、快適にご乗車頂ける様、6月中旬から長橋線に<strong>順次弱冷車を導入</strong>してまいります。是非、ご利用ください。',
  'お体の不自由な方等が駅構内で困っているときには、皆様の一声が明るい社会を作ります。ご理解とご協力をお願いします。',
];
class PlatformDisplayRow extends HTMLElement {
  type;
  typeElem;
  index;
  indexElem;
  time;
  timeElem;
  track;
  trackElem;
  dir;
  dirElem;
  constructor() {
    super();
    this.shadow = this.attachShadow({ mode: "open" });
  }
  connectedCallback() {
    const wrapper = document.createElement("div");
    this.shadow.appendChild(wrapper);
    const row = document.createElement("tr");
    wrapper.appendChild(row);
    this.typeElem = document.createElement("td");
    this.typeElem.classList.add("type");
    this.typeElem.textContent = this.type;
    row.appendChild(this.typeElem);
    this.indexElem = document.createElement("td");
    this.indexElem.classList.add("index");
    this.indexElem.textContent = this.index;
    row.appendChild(this.indexElem);
    this.timeElem = document.createElement("td");
    this.timeElem.classList.add("time");
    this.timeElem.textContent = this.time;
    row.appendChild(this.timeElem);
    this.trackElem = document.createElement("td");
    this.trackElem.classList.add("track");
    this.trackElem.textContent = this.track;
    row.appendChild(this.trackElem);
    this.dirElem = document.createElement("td");
    this.dirElem.classList.add("dir");
    this.dirElem.textContent = this.dir;
    row.appendChild(this.dirElem);
  }

  static observedAttributes = [ "type", "index", "time", "track", "dir", ];
  attributeChangedCallback(name, oldValue, newValue) {
    const directAttributes = [ "type", "index", "time", "track", "dir", ];
    if (directAttributes.includes(name)) {
      this[name] = newValue;
      if (this[name+"Elem"]) {
        this[name+"Elem"].textContent = this[name];
      }
    }
  }
}

class PlatformDisplay extends HTMLElement {
  constructor() {
    super();
    this.shadow = this.attachShadow({ mode: "open" });
    this.scrollIndex = 0;
  }
  static observedAttributes = [ "allocs" ];
  attributeChangedCallback(name, oldValue, newValue) {
    if (name == "allocs") {
      this.allocsData = JSON.parse(newValue);
      this.updateAllocs();
    }
  }
  allocs;
  allocsData;
  wrapper;
  clock;
  scroll;
  updateAllocs() {
    if (!this.wrapper) return;
    this.allocs.textContent = '';
    const header = document.createElement("tr");
    const newHeader = (name) => {
      const elem = document.createElement("th");
      elem.textContent = name;
      return elem;
    };
    header.appendChild(newHeader("番目"));
    header.appendChild(newHeader("種別"));
    header.appendChild(newHeader("運用"));
    header.appendChild(newHeader("到着時刻"));
    header.appendChild(newHeader("のりば"));
    header.appendChild(newHeader("行先"));
    this.allocs.appendChild(header);
    const newCell = (type, name) => {
      const elem = document.createElement("td");
      elem.textContent = name;
      elem.classList.add(type);
      elem.dataset.value = name;
      return elem;
    };
    for (let i of this.allocsData.keys()) {
      const alloc = this.allocsData[i];
      const row = document.createElement("tr");
      const iNames = ["先発", "次発"];
      row.appendChild(newCell("i", (i < iNames.length) ? iNames[i] : `${i+1}`));
      row.appendChild(newCell("type", alloc.type));
      row.appendChild(newCell("index", alloc.index));
      row.appendChild(newCell("time", alloc.time));
      row.appendChild(newCell("track", alloc.track));
      row.appendChild(newCell("dir", alloc.dir));
      this.allocs.appendChild(row);
    }
  }
  updateClock() {
    if (!this.clock) return;
    const d = new Date();
    const h = d.getHours().toString().padStart(2, '0');
    const m = d.getMinutes().toString().padStart(2, '0');
    const s = d.getSeconds().toString().padStart(2, '0');
    this.clock.textContent = `${h}:${m}:${s}`;
  }
  iterateScroll() {
    this.scrollIndex = (this.scrollIndex + 1) % scrolls.length;
    this.scroll.innerHTML = scrolls[this.scrollIndex];
  }
  connectedCallback() {
    this.wrapper = document.createElement("div");
    this.wrapper.classList.add("wrapper");

    const place = document.createElement("div");
    place.classList.add("place");
    this.wrapper.appendChild(place);
    const num = document.createElement("span");
    num.classList.add("num");
    num.textContent = "1";
    place.appendChild(num)
    const name = document.createElement("div");
    name.classList.add("name");
    name.textContent = "長橋線";
    place.appendChild(name);
    const clockContainer = document.createElement("div");
    place.appendChild(clockContainer);
    const clockLabel = document.createElement("span");
    clockLabel.classList.add("clock-label");
    clockLabel.textContent = "現在時刻";
    clockContainer.appendChild(clockLabel);
    this.clock = document.createElement("div");
    this.clock.classList.add("clock");
    this.clock.textContent = "--:--:--";
    clockContainer.appendChild(this.clock);
    let this2 = this;
    setInterval(function() {this2.updateClock()}, 1000);

    this.allocs = document.createElement("table");
    this.allocs.classList.add("allocs");
    this.wrapper.appendChild(this.allocs);
    this.updateAllocs();

    this.scroll = document.createElement("div");
    this.wrapper.appendChild(this.scroll);
    this.scroll.classList.add("scroll");
    this.scroll.innerHTML = tenji;
    this.scroll.addEventListener("animationiteration", function() {
      this2.iterateScroll();
    }, false);

    const style = document.createElement("style");
    style.textContent = styleText;
    this.shadow.appendChild(style);
    this.shadow.appendChild(this.wrapper);
  }
}

customElements.define("platform-display", PlatformDisplay);
customElements.define("platform-display-row", PlatformDisplayRow);

(function polyfill() {
  const relList = document.createElement("link").relList;
  if (relList && relList.supports && relList.supports("modulepreload")) {
    return;
  }
  for (const link of document.querySelectorAll('link[rel="modulepreload"]')) {
    processPreload(link);
  }
  new MutationObserver((mutations) => {
    for (const mutation of mutations) {
      if (mutation.type !== "childList") {
        continue;
      }
      for (const node of mutation.addedNodes) {
        if (node.tagName === "LINK" && node.rel === "modulepreload")
          processPreload(node);
      }
    }
  }).observe(document, { childList: true, subtree: true });
  function getFetchOpts(link) {
    const fetchOpts = {};
    if (link.integrity) fetchOpts.integrity = link.integrity;
    if (link.referrerPolicy) fetchOpts.referrerPolicy = link.referrerPolicy;
    if (link.crossOrigin === "use-credentials")
      fetchOpts.credentials = "include";
    else if (link.crossOrigin === "anonymous") fetchOpts.credentials = "omit";
    else fetchOpts.credentials = "same-origin";
    return fetchOpts;
  }
  function processPreload(link) {
    if (link.ep)
      return;
    link.ep = true;
    const fetchOpts = getFetchOpts(link);
    fetch(link.href, fetchOpts);
  }
})();
/**
* @vue/shared v3.5.38
* (c) 2018-present Yuxi (Evan) You and Vue contributors
* @license MIT
**/
// @__NO_SIDE_EFFECTS__
function makeMap(str) {
  const map = /* @__PURE__ */ Object.create(null);
  for (const key of str.split(",")) map[key] = 1;
  return (val) => val in map;
}
const EMPTY_OBJ = {};
const EMPTY_ARR = [];
const NOOP = () => {
};
const NO = () => false;
const isOn = (key) => key.charCodeAt(0) === 111 && key.charCodeAt(1) === 110 && // uppercase letter
(key.charCodeAt(2) > 122 || key.charCodeAt(2) < 97);
const isModelListener = (key) => key.startsWith("onUpdate:");
const extend = Object.assign;
const remove = (arr, el) => {
  const i = arr.indexOf(el);
  if (i > -1) {
    arr.splice(i, 1);
  }
};
const hasOwnProperty$2 = Object.prototype.hasOwnProperty;
const hasOwn$1 = (val, key) => hasOwnProperty$2.call(val, key);
const isArray$1 = Array.isArray;
const isMap = (val) => toTypeString$1(val) === "[object Map]";
const isSet = (val) => toTypeString$1(val) === "[object Set]";
const isDate$1 = (val) => toTypeString$1(val) === "[object Date]";
const isFunction$1 = (val) => typeof val === "function";
const isString$2 = (val) => typeof val === "string";
const isSymbol = (val) => typeof val === "symbol";
const isObject$2 = (val) => val !== null && typeof val === "object";
const isPromise$1 = (val) => {
  return (isObject$2(val) || isFunction$1(val)) && isFunction$1(val.then) && isFunction$1(val.catch);
};
const objectToString$1 = Object.prototype.toString;
const toTypeString$1 = (value) => objectToString$1.call(value);
const toRawType = (value) => {
  return toTypeString$1(value).slice(8, -1);
};
const isPlainObject$2 = (val) => toTypeString$1(val) === "[object Object]";
const isIntegerKey = (key) => isString$2(key) && key !== "NaN" && key[0] !== "-" && "" + parseInt(key, 10) === key;
const isReservedProp = /* @__PURE__ */ makeMap(
  // the leading comma is intentional so empty string "" is also included
  ",key,ref,ref_for,ref_key,onVnodeBeforeMount,onVnodeMounted,onVnodeBeforeUpdate,onVnodeUpdated,onVnodeBeforeUnmount,onVnodeUnmounted"
);
const cacheStringFunction = (fn) => {
  const cache2 = /* @__PURE__ */ Object.create(null);
  return (str) => {
    const hit = cache2[str];
    return hit || (cache2[str] = fn(str));
  };
};
const camelizeRE = /-\w/g;
const camelize = cacheStringFunction(
  (str) => {
    return str.replace(camelizeRE, (c) => c.slice(1).toUpperCase());
  }
);
const hyphenateRE = /\B([A-Z])/g;
const hyphenate = cacheStringFunction(
  (str) => str.replace(hyphenateRE, "-$1").toLowerCase()
);
const capitalize$1 = cacheStringFunction((str) => {
  return str.charAt(0).toUpperCase() + str.slice(1);
});
const toHandlerKey = cacheStringFunction(
  (str) => {
    const s = str ? `on${capitalize$1(str)}` : ``;
    return s;
  }
);
const hasChanged = (value, oldValue) => !Object.is(value, oldValue);
const invokeArrayFns = (fns, ...arg) => {
  for (let i = 0; i < fns.length; i++) {
    fns[i](...arg);
  }
};
const def = (obj, key, value, writable = false) => {
  Object.defineProperty(obj, key, {
    configurable: true,
    enumerable: false,
    writable,
    value
  });
};
const looseToNumber = (val) => {
  const n = parseFloat(val);
  return isNaN(n) ? val : n;
};
const toNumber = (val) => {
  const n = isString$2(val) ? Number(val) : NaN;
  return isNaN(n) ? val : n;
};
let _globalThis$1;
const getGlobalThis$1 = () => {
  return _globalThis$1 || (_globalThis$1 = typeof globalThis !== "undefined" ? globalThis : typeof self !== "undefined" ? self : typeof window !== "undefined" ? window : typeof global !== "undefined" ? global : {});
};
function normalizeStyle(value) {
  if (isArray$1(value)) {
    const res = {};
    for (let i = 0; i < value.length; i++) {
      const item = value[i];
      const normalized = isString$2(item) ? parseStringStyle(item) : normalizeStyle(item);
      if (normalized) {
        for (const key in normalized) {
          res[key] = normalized[key];
        }
      }
    }
    return res;
  } else if (isString$2(value) || isObject$2(value)) {
    return value;
  }
}
const listDelimiterRE = /;(?![^(]*\))/g;
const propertyDelimiterRE = /:([^]+)/;
const styleCommentRE = /\/\*[^]*?\*\//g;
function parseStringStyle(cssText) {
  const ret = {};
  cssText.replace(styleCommentRE, "").split(listDelimiterRE).forEach((item) => {
    if (item) {
      const tmp = item.split(propertyDelimiterRE);
      tmp.length > 1 && (ret[tmp[0].trim()] = tmp[1].trim());
    }
  });
  return ret;
}
function normalizeClass(value) {
  let res = "";
  if (isString$2(value)) {
    res = value;
  } else if (isArray$1(value)) {
    for (let i = 0; i < value.length; i++) {
      const normalized = normalizeClass(value[i]);
      if (normalized) {
        res += normalized + " ";
      }
    }
  } else if (isObject$2(value)) {
    for (const name in value) {
      if (value[name]) {
        res += name + " ";
      }
    }
  }
  return res.trim();
}
const specialBooleanAttrs = `itemscope,allowfullscreen,formnovalidate,ismap,nomodule,novalidate,readonly`;
const isSpecialBooleanAttr = /* @__PURE__ */ makeMap(specialBooleanAttrs);
function includeBooleanAttr(value) {
  return !!value || value === "";
}
function looseCompareArrays(a, b) {
  if (a.length !== b.length) return false;
  let equal = true;
  for (let i = 0; equal && i < a.length; i++) {
    equal = looseEqual(a[i], b[i]);
  }
  return equal;
}
function looseEqual(a, b) {
  if (a === b) return true;
  let aValidType = isDate$1(a);
  let bValidType = isDate$1(b);
  if (aValidType || bValidType) {
    return aValidType && bValidType ? a.getTime() === b.getTime() : false;
  }
  aValidType = isSymbol(a);
  bValidType = isSymbol(b);
  if (aValidType || bValidType) {
    return a === b;
  }
  aValidType = isArray$1(a);
  bValidType = isArray$1(b);
  if (aValidType || bValidType) {
    return aValidType && bValidType ? looseCompareArrays(a, b) : false;
  }
  aValidType = isObject$2(a);
  bValidType = isObject$2(b);
  if (aValidType || bValidType) {
    if (!aValidType || !bValidType) {
      return false;
    }
    const aKeysCount = Object.keys(a).length;
    const bKeysCount = Object.keys(b).length;
    if (aKeysCount !== bKeysCount) {
      return false;
    }
    for (const key in a) {
      const aHasKey = a.hasOwnProperty(key);
      const bHasKey = b.hasOwnProperty(key);
      if (aHasKey && !bHasKey || !aHasKey && bHasKey || !looseEqual(a[key], b[key])) {
        return false;
      }
    }
  }
  return String(a) === String(b);
}
function looseIndexOf(arr, val) {
  return arr.findIndex((item) => looseEqual(item, val));
}
const isRef$1 = (val) => {
  return !!(val && val["__v_isRef"] === true);
};
const toDisplayString$1 = (val) => {
  return isString$2(val) ? val : val == null ? "" : isArray$1(val) || isObject$2(val) && (val.toString === objectToString$1 || !isFunction$1(val.toString)) ? isRef$1(val) ? toDisplayString$1(val.value) : JSON.stringify(val, replacer, 2) : String(val);
};
const replacer = (_key, val) => {
  if (isRef$1(val)) {
    return replacer(_key, val.value);
  } else if (isMap(val)) {
    return {
      [`Map(${val.size})`]: [...val.entries()].reduce(
        (entries, [key, val2], i) => {
          entries[stringifySymbol(key, i) + " =>"] = val2;
          return entries;
        },
        {}
      )
    };
  } else if (isSet(val)) {
    return {
      [`Set(${val.size})`]: [...val.values()].map((v) => stringifySymbol(v))
    };
  } else if (isSymbol(val)) {
    return stringifySymbol(val);
  } else if (isObject$2(val) && !isArray$1(val) && !isPlainObject$2(val)) {
    return String(val);
  }
  return val;
};
const stringifySymbol = (v, i = "") => {
  var _a;
  return (
    // Symbol.description in es2019+ so we need to cast here to pass
    // the lib: es2016 check
    isSymbol(v) ? `Symbol(${(_a = v.description) != null ? _a : i})` : v
  );
};
/**
* @vue/reactivity v3.5.38
* (c) 2018-present Yuxi (Evan) You and Vue contributors
* @license MIT
**/
let activeEffectScope;
class EffectScope {
  // TODO isolatedDeclarations "__v_skip"
  constructor(detached = false) {
    this.detached = detached;
    this._active = true;
    this._on = 0;
    this.effects = [];
    this.cleanups = [];
    this._isPaused = false;
    this._warnOnRun = true;
    this.__v_skip = true;
    if (!detached && activeEffectScope) {
      if (activeEffectScope.active) {
        this.parent = activeEffectScope;
        this.index = (activeEffectScope.scopes || (activeEffectScope.scopes = [])).push(
          this
        ) - 1;
      } else {
        this._active = false;
        this._warnOnRun = false;
      }
    }
  }
  get active() {
    return this._active;
  }
  pause() {
    if (this._active) {
      this._isPaused = true;
      let i, l;
      if (this.scopes) {
        for (i = 0, l = this.scopes.length; i < l; i++) {
          this.scopes[i].pause();
        }
      }
      for (i = 0, l = this.effects.length; i < l; i++) {
        this.effects[i].pause();
      }
    }
  }
  /**
   * Resumes the effect scope, including all child scopes and effects.
   */
  resume() {
    if (this._active) {
      if (this._isPaused) {
        this._isPaused = false;
        let i, l;
        if (this.scopes) {
          for (i = 0, l = this.scopes.length; i < l; i++) {
            this.scopes[i].resume();
          }
        }
        for (i = 0, l = this.effects.length; i < l; i++) {
          this.effects[i].resume();
        }
      }
    }
  }
  run(fn) {
    if (this._active) {
      const currentEffectScope = activeEffectScope;
      try {
        activeEffectScope = this;
        return fn();
      } finally {
        activeEffectScope = currentEffectScope;
      }
    }
  }
  /**
   * This should only be called on non-detached scopes
   * @internal
   */
  on() {
    if (++this._on === 1) {
      this.prevScope = activeEffectScope;
      activeEffectScope = this;
    }
  }
  /**
   * This should only be called on non-detached scopes
   * @internal
   */
  off() {
    if (this._on > 0 && --this._on === 0) {
      if (activeEffectScope === this) {
        activeEffectScope = this.prevScope;
      } else {
        let current = activeEffectScope;
        while (current) {
          if (current.prevScope === this) {
            current.prevScope = this.prevScope;
            break;
          }
          current = current.prevScope;
        }
      }
      this.prevScope = void 0;
    }
  }
  stop(fromParent) {
    if (this._active) {
      this._active = false;
      let i, l;
      for (i = 0, l = this.effects.length; i < l; i++) {
        this.effects[i].stop();
      }
      this.effects.length = 0;
      for (i = 0, l = this.cleanups.length; i < l; i++) {
        this.cleanups[i]();
      }
      this.cleanups.length = 0;
      if (this.scopes) {
        for (i = 0, l = this.scopes.length; i < l; i++) {
          this.scopes[i].stop(true);
        }
        this.scopes.length = 0;
      }
      if (!this.detached && this.parent && !fromParent) {
        const last = this.parent.scopes.pop();
        if (last && last !== this) {
          this.parent.scopes[this.index] = last;
          last.index = this.index;
        }
      }
      this.parent = void 0;
    }
  }
}
function effectScope(detached) {
  return new EffectScope(detached);
}
function getCurrentScope() {
  return activeEffectScope;
}
function onScopeDispose(fn, failSilently = false) {
  if (activeEffectScope) {
    activeEffectScope.cleanups.push(fn);
  }
}
let activeSub;
const pausedQueueEffects = /* @__PURE__ */ new WeakSet();
class ReactiveEffect {
  constructor(fn) {
    this.fn = fn;
    this.deps = void 0;
    this.depsTail = void 0;
    this.flags = 1 | 4;
    this.next = void 0;
    this.cleanup = void 0;
    this.scheduler = void 0;
    if (activeEffectScope) {
      if (activeEffectScope.active) {
        activeEffectScope.effects.push(this);
      } else {
        this.flags &= -2;
      }
    }
  }
  pause() {
    this.flags |= 64;
  }
  resume() {
    if (this.flags & 64) {
      this.flags &= -65;
      if (pausedQueueEffects.has(this)) {
        pausedQueueEffects.delete(this);
        this.trigger();
      }
    }
  }
  /**
   * @internal
   */
  notify() {
    if (this.flags & 2 && !(this.flags & 32)) {
      return;
    }
    if (!(this.flags & 8)) {
      batch(this);
    }
  }
  run() {
    if (!(this.flags & 1)) {
      return this.fn();
    }
    this.flags |= 2;
    cleanupEffect(this);
    prepareDeps(this);
    const prevEffect = activeSub;
    const prevShouldTrack = shouldTrack;
    activeSub = this;
    shouldTrack = true;
    try {
      return this.fn();
    } finally {
      cleanupDeps(this);
      activeSub = prevEffect;
      shouldTrack = prevShouldTrack;
      this.flags &= -3;
    }
  }
  stop() {
    if (this.flags & 1) {
      for (let link = this.deps; link; link = link.nextDep) {
        removeSub(link);
      }
      this.deps = this.depsTail = void 0;
      cleanupEffect(this);
      this.onStop && this.onStop();
      this.flags &= -2;
    }
  }
  trigger() {
    if (this.flags & 64) {
      pausedQueueEffects.add(this);
    } else if (this.scheduler) {
      this.scheduler();
    } else {
      this.runIfDirty();
    }
  }
  /**
   * @internal
   */
  runIfDirty() {
    if (isDirty(this)) {
      this.run();
    }
  }
  get dirty() {
    return isDirty(this);
  }
}
let batchDepth = 0;
let batchedSub;
let batchedComputed;
function batch(sub, isComputed2 = false) {
  sub.flags |= 8;
  if (isComputed2) {
    sub.next = batchedComputed;
    batchedComputed = sub;
    return;
  }
  sub.next = batchedSub;
  batchedSub = sub;
}
function startBatch() {
  batchDepth++;
}
function endBatch() {
  if (--batchDepth > 0) {
    return;
  }
  if (batchedComputed) {
    let e = batchedComputed;
    batchedComputed = void 0;
    while (e) {
      const next = e.next;
      e.next = void 0;
      e.flags &= -9;
      e = next;
    }
  }
  let error;
  while (batchedSub) {
    let e = batchedSub;
    batchedSub = void 0;
    while (e) {
      const next = e.next;
      e.next = void 0;
      e.flags &= -9;
      if (e.flags & 1) {
        try {
          ;
          e.trigger();
        } catch (err) {
          if (!error) error = err;
        }
      }
      e = next;
    }
  }
  if (error) throw error;
}
function prepareDeps(sub) {
  for (let link = sub.deps; link; link = link.nextDep) {
    link.version = -1;
    link.prevActiveLink = link.dep.activeLink;
    link.dep.activeLink = link;
  }
}
function cleanupDeps(sub) {
  let head;
  let tail = sub.depsTail;
  let link = tail;
  while (link) {
    const prev = link.prevDep;
    if (link.version === -1) {
      if (link === tail) tail = prev;
      removeSub(link);
      removeDep(link);
    } else {
      head = link;
    }
    link.dep.activeLink = link.prevActiveLink;
    link.prevActiveLink = void 0;
    link = prev;
  }
  sub.deps = head;
  sub.depsTail = tail;
}
function isDirty(sub) {
  for (let link = sub.deps; link; link = link.nextDep) {
    if (link.dep.version !== link.version || link.dep.computed && (refreshComputed(link.dep.computed) || link.dep.version !== link.version)) {
      return true;
    }
  }
  if (sub._dirty) {
    return true;
  }
  return false;
}
function refreshComputed(computed2) {
  if (computed2.flags & 4 && !(computed2.flags & 16)) {
    return;
  }
  computed2.flags &= -17;
  if (computed2.globalVersion === globalVersion) {
    return;
  }
  computed2.globalVersion = globalVersion;
  if (!computed2.isSSR && computed2.flags & 128 && (!computed2.deps && !computed2._dirty || !isDirty(computed2))) {
    return;
  }
  computed2.flags |= 2;
  const dep = computed2.dep;
  const prevSub = activeSub;
  const prevShouldTrack = shouldTrack;
  activeSub = computed2;
  shouldTrack = true;
  try {
    prepareDeps(computed2);
    const value = computed2.fn(computed2._value);
    if (dep.version === 0 || hasChanged(value, computed2._value)) {
      computed2.flags |= 128;
      computed2._value = value;
      dep.version++;
    }
  } catch (err) {
    dep.version++;
    throw err;
  } finally {
    activeSub = prevSub;
    shouldTrack = prevShouldTrack;
    cleanupDeps(computed2);
    computed2.flags &= -3;
  }
}
function removeSub(link, soft = false) {
  const { dep, prevSub, nextSub } = link;
  if (prevSub) {
    prevSub.nextSub = nextSub;
    link.prevSub = void 0;
  }
  if (nextSub) {
    nextSub.prevSub = prevSub;
    link.nextSub = void 0;
  }
  if (dep.subs === link) {
    dep.subs = prevSub;
    if (!prevSub && dep.computed) {
      dep.computed.flags &= -5;
      for (let l = dep.computed.deps; l; l = l.nextDep) {
        removeSub(l, true);
      }
    }
  }
  if (!soft && !--dep.sc && dep.map) {
    dep.map.delete(dep.key);
  }
}
function removeDep(link) {
  const { prevDep, nextDep } = link;
  if (prevDep) {
    prevDep.nextDep = nextDep;
    link.prevDep = void 0;
  }
  if (nextDep) {
    nextDep.prevDep = prevDep;
    link.nextDep = void 0;
  }
}
let shouldTrack = true;
const trackStack = [];
function pauseTracking() {
  trackStack.push(shouldTrack);
  shouldTrack = false;
}
function resetTracking() {
  const last = trackStack.pop();
  shouldTrack = last === void 0 ? true : last;
}
function cleanupEffect(e) {
  const { cleanup } = e;
  e.cleanup = void 0;
  if (cleanup) {
    const prevSub = activeSub;
    activeSub = void 0;
    try {
      cleanup();
    } finally {
      activeSub = prevSub;
    }
  }
}
let globalVersion = 0;
class Link {
  constructor(sub, dep) {
    this.sub = sub;
    this.dep = dep;
    this.version = dep.version;
    this.nextDep = this.prevDep = this.nextSub = this.prevSub = this.prevActiveLink = void 0;
  }
}
class Dep {
  // TODO isolatedDeclarations "__v_skip"
  constructor(computed2) {
    this.computed = computed2;
    this.version = 0;
    this.activeLink = void 0;
    this.subs = void 0;
    this.map = void 0;
    this.key = void 0;
    this.sc = 0;
    this.__v_skip = true;
  }
  track(debugInfo) {
    if (!activeSub || !shouldTrack || activeSub === this.computed) {
      return;
    }
    let link = this.activeLink;
    if (link === void 0 || link.sub !== activeSub) {
      link = this.activeLink = new Link(activeSub, this);
      if (!activeSub.deps) {
        activeSub.deps = activeSub.depsTail = link;
      } else {
        link.prevDep = activeSub.depsTail;
        activeSub.depsTail.nextDep = link;
        activeSub.depsTail = link;
      }
      addSub(link);
    } else if (link.version === -1) {
      link.version = this.version;
      if (link.nextDep) {
        const next = link.nextDep;
        next.prevDep = link.prevDep;
        if (link.prevDep) {
          link.prevDep.nextDep = next;
        }
        link.prevDep = activeSub.depsTail;
        link.nextDep = void 0;
        activeSub.depsTail.nextDep = link;
        activeSub.depsTail = link;
        if (activeSub.deps === link) {
          activeSub.deps = next;
        }
      }
    }
    return link;
  }
  trigger(debugInfo) {
    this.version++;
    globalVersion++;
    this.notify(debugInfo);
  }
  notify(debugInfo) {
    startBatch();
    try {
      if (false) ;
      for (let link = this.subs; link; link = link.prevSub) {
        if (link.sub.notify()) {
          ;
          link.sub.dep.notify();
        }
      }
    } finally {
      endBatch();
    }
  }
}
function addSub(link) {
  link.dep.sc++;
  if (link.sub.flags & 4) {
    const computed2 = link.dep.computed;
    if (computed2 && !link.dep.subs) {
      computed2.flags |= 4 | 16;
      for (let l = computed2.deps; l; l = l.nextDep) {
        addSub(l);
      }
    }
    const currentTail = link.dep.subs;
    if (currentTail !== link) {
      link.prevSub = currentTail;
      if (currentTail) currentTail.nextSub = link;
    }
    link.dep.subs = link;
  }
}
const targetMap = /* @__PURE__ */ new WeakMap();
const ITERATE_KEY = /* @__PURE__ */ Symbol(
  ""
);
const MAP_KEY_ITERATE_KEY = /* @__PURE__ */ Symbol(
  ""
);
const ARRAY_ITERATE_KEY = /* @__PURE__ */ Symbol(
  ""
);
function track(target, type, key) {
  if (shouldTrack && activeSub) {
    let depsMap = targetMap.get(target);
    if (!depsMap) {
      targetMap.set(target, depsMap = /* @__PURE__ */ new Map());
    }
    let dep = depsMap.get(key);
    if (!dep) {
      depsMap.set(key, dep = new Dep());
      dep.map = depsMap;
      dep.key = key;
    }
    {
      dep.track();
    }
  }
}
function trigger(target, type, key, newValue, oldValue, oldTarget) {
  const depsMap = targetMap.get(target);
  if (!depsMap) {
    globalVersion++;
    return;
  }
  const run = (dep) => {
    if (dep) {
      {
        dep.trigger();
      }
    }
  };
  startBatch();
  if (type === "clear") {
    depsMap.forEach(run);
  } else {
    const targetIsArray = isArray$1(target);
    const isArrayIndex = targetIsArray && isIntegerKey(key);
    if (targetIsArray && key === "length") {
      const newLength = Number(newValue);
      depsMap.forEach((dep, key2) => {
        if (key2 === "length" || key2 === ARRAY_ITERATE_KEY || !isSymbol(key2) && key2 >= newLength) {
          run(dep);
        }
      });
    } else {
      if (key !== void 0 || depsMap.has(void 0)) {
        run(depsMap.get(key));
      }
      if (isArrayIndex) {
        run(depsMap.get(ARRAY_ITERATE_KEY));
      }
      switch (type) {
        case "add":
          if (!targetIsArray) {
            run(depsMap.get(ITERATE_KEY));
            if (isMap(target)) {
              run(depsMap.get(MAP_KEY_ITERATE_KEY));
            }
          } else if (isArrayIndex) {
            run(depsMap.get("length"));
          }
          break;
        case "delete":
          if (!targetIsArray) {
            run(depsMap.get(ITERATE_KEY));
            if (isMap(target)) {
              run(depsMap.get(MAP_KEY_ITERATE_KEY));
            }
          }
          break;
        case "set":
          if (isMap(target)) {
            run(depsMap.get(ITERATE_KEY));
          }
          break;
      }
    }
  }
  endBatch();
}
function getDepFromReactive(object, key) {
  const depMap = targetMap.get(object);
  return depMap && depMap.get(key);
}
function reactiveReadArray(array) {
  const raw = /* @__PURE__ */ toRaw(array);
  if (raw === array) return raw;
  track(raw, "iterate", ARRAY_ITERATE_KEY);
  return /* @__PURE__ */ isShallow(array) ? raw : raw.map(toReactive);
}
function shallowReadArray(arr) {
  track(arr = /* @__PURE__ */ toRaw(arr), "iterate", ARRAY_ITERATE_KEY);
  return arr;
}
function toWrapped(target, item) {
  if (/* @__PURE__ */ isReadonly(target)) {
    return /* @__PURE__ */ isReactive(target) ? toReadonly(toReactive(item)) : toReadonly(item);
  }
  return toReactive(item);
}
const arrayInstrumentations = {
  __proto__: null,
  [Symbol.iterator]() {
    return iterator(this, Symbol.iterator, (item) => toWrapped(this, item));
  },
  concat(...args) {
    return reactiveReadArray(this).concat(
      ...args.map((x) => isArray$1(x) ? reactiveReadArray(x) : x)
    );
  },
  entries() {
    return iterator(this, "entries", (value) => {
      value[1] = toWrapped(this, value[1]);
      return value;
    });
  },
  every(fn, thisArg) {
    return apply$1(this, "every", fn, thisArg, void 0, arguments);
  },
  filter(fn, thisArg) {
    return apply$1(
      this,
      "filter",
      fn,
      thisArg,
      (v) => v.map((item) => toWrapped(this, item)),
      arguments
    );
  },
  find(fn, thisArg) {
    return apply$1(
      this,
      "find",
      fn,
      thisArg,
      (item) => toWrapped(this, item),
      arguments
    );
  },
  findIndex(fn, thisArg) {
    return apply$1(this, "findIndex", fn, thisArg, void 0, arguments);
  },
  findLast(fn, thisArg) {
    return apply$1(
      this,
      "findLast",
      fn,
      thisArg,
      (item) => toWrapped(this, item),
      arguments
    );
  },
  findLastIndex(fn, thisArg) {
    return apply$1(this, "findLastIndex", fn, thisArg, void 0, arguments);
  },
  // flat, flatMap could benefit from ARRAY_ITERATE but are not straight-forward to implement
  forEach(fn, thisArg) {
    return apply$1(this, "forEach", fn, thisArg, void 0, arguments);
  },
  includes(...args) {
    return searchProxy(this, "includes", args);
  },
  indexOf(...args) {
    return searchProxy(this, "indexOf", args);
  },
  join(separator2) {
    return reactiveReadArray(this).join(separator2);
  },
  // keys() iterator only reads `length`, no optimization required
  lastIndexOf(...args) {
    return searchProxy(this, "lastIndexOf", args);
  },
  map(fn, thisArg) {
    return apply$1(this, "map", fn, thisArg, void 0, arguments);
  },
  pop() {
    return noTracking(this, "pop");
  },
  push(...args) {
    return noTracking(this, "push", args);
  },
  reduce(fn, ...args) {
    return reduce(this, "reduce", fn, args);
  },
  reduceRight(fn, ...args) {
    return reduce(this, "reduceRight", fn, args);
  },
  shift() {
    return noTracking(this, "shift");
  },
  // slice could use ARRAY_ITERATE but also seems to beg for range tracking
  some(fn, thisArg) {
    return apply$1(this, "some", fn, thisArg, void 0, arguments);
  },
  splice(...args) {
    return noTracking(this, "splice", args);
  },
  toReversed() {
    return reactiveReadArray(this).toReversed();
  },
  toSorted(comparer) {
    return reactiveReadArray(this).toSorted(comparer);
  },
  toSpliced(...args) {
    return reactiveReadArray(this).toSpliced(...args);
  },
  unshift(...args) {
    return noTracking(this, "unshift", args);
  },
  values() {
    return iterator(this, "values", (item) => toWrapped(this, item));
  }
};
function iterator(self2, method, wrapValue) {
  const arr = shallowReadArray(self2);
  const iter = arr[method]();
  if (arr !== self2 && !/* @__PURE__ */ isShallow(self2)) {
    iter._next = iter.next;
    iter.next = () => {
      const result = iter._next();
      if (!result.done) {
        result.value = wrapValue(result.value);
      }
      return result;
    };
  }
  return iter;
}
const arrayProto = Array.prototype;
function apply$1(self2, method, fn, thisArg, wrappedRetFn, args) {
  const arr = shallowReadArray(self2);
  const needsWrap = arr !== self2 && !/* @__PURE__ */ isShallow(self2);
  const methodFn = arr[method];
  if (methodFn !== arrayProto[method]) {
    const result2 = methodFn.apply(self2, args);
    return needsWrap ? toReactive(result2) : result2;
  }
  let wrappedFn = fn;
  if (arr !== self2) {
    if (needsWrap) {
      wrappedFn = function(item, index) {
        return fn.call(this, toWrapped(self2, item), index, self2);
      };
    } else if (fn.length > 2) {
      wrappedFn = function(item, index) {
        return fn.call(this, item, index, self2);
      };
    }
  }
  const result = methodFn.call(arr, wrappedFn, thisArg);
  return needsWrap && wrappedRetFn ? wrappedRetFn(result) : result;
}
function reduce(self2, method, fn, args) {
  const arr = shallowReadArray(self2);
  const needsWrap = arr !== self2 && !/* @__PURE__ */ isShallow(self2);
  let wrappedFn = fn;
  let wrapInitialAccumulator = false;
  if (arr !== self2) {
    if (needsWrap) {
      wrapInitialAccumulator = args.length === 0;
      wrappedFn = function(acc, item, index) {
        if (wrapInitialAccumulator) {
          wrapInitialAccumulator = false;
          acc = toWrapped(self2, acc);
        }
        return fn.call(this, acc, toWrapped(self2, item), index, self2);
      };
    } else if (fn.length > 3) {
      wrappedFn = function(acc, item, index) {
        return fn.call(this, acc, item, index, self2);
      };
    }
  }
  const result = arr[method](wrappedFn, ...args);
  return wrapInitialAccumulator ? toWrapped(self2, result) : result;
}
function searchProxy(self2, method, args) {
  const arr = /* @__PURE__ */ toRaw(self2);
  track(arr, "iterate", ARRAY_ITERATE_KEY);
  const res = arr[method](...args);
  if ((res === -1 || res === false) && /* @__PURE__ */ isProxy(args[0])) {
    args[0] = /* @__PURE__ */ toRaw(args[0]);
    return arr[method](...args);
  }
  return res;
}
function noTracking(self2, method, args = []) {
  pauseTracking();
  startBatch();
  const res = (/* @__PURE__ */ toRaw(self2))[method].apply(self2, args);
  endBatch();
  resetTracking();
  return res;
}
const isNonTrackableKeys = /* @__PURE__ */ makeMap(`__proto__,__v_isRef,__isVue`);
const builtInSymbols = new Set(
  /* @__PURE__ */ Object.getOwnPropertyNames(Symbol).filter((key) => key !== "arguments" && key !== "caller").map((key) => Symbol[key]).filter(isSymbol)
);
function hasOwnProperty$1(key) {
  if (!isSymbol(key)) key = String(key);
  const obj = /* @__PURE__ */ toRaw(this);
  track(obj, "has", key);
  return obj.hasOwnProperty(key);
}
class BaseReactiveHandler {
  constructor(_isReadonly = false, _isShallow = false) {
    this._isReadonly = _isReadonly;
    this._isShallow = _isShallow;
  }
  get(target, key, receiver) {
    if (key === "__v_skip") return target["__v_skip"];
    const isReadonly2 = this._isReadonly, isShallow2 = this._isShallow;
    if (key === "__v_isReactive") {
      return !isReadonly2;
    } else if (key === "__v_isReadonly") {
      return isReadonly2;
    } else if (key === "__v_isShallow") {
      return isShallow2;
    } else if (key === "__v_raw") {
      if (receiver === (isReadonly2 ? isShallow2 ? shallowReadonlyMap : readonlyMap : isShallow2 ? shallowReactiveMap : reactiveMap).get(target) || // receiver is not the reactive proxy, but has the same prototype
      // this means the receiver is a user proxy of the reactive proxy
      Object.getPrototypeOf(target) === Object.getPrototypeOf(receiver)) {
        return target;
      }
      return;
    }
    const targetIsArray = isArray$1(target);
    if (!isReadonly2) {
      let fn;
      if (targetIsArray && (fn = arrayInstrumentations[key])) {
        return fn;
      }
      if (key === "hasOwnProperty") {
        return hasOwnProperty$1;
      }
    }
    const res = Reflect.get(
      target,
      key,
      // if this is a proxy wrapping a ref, return methods using the raw ref
      // as receiver so that we don't have to call `toRaw` on the ref in all
      // its class methods
      /* @__PURE__ */ isRef(target) ? target : receiver
    );
    if (isSymbol(key) ? builtInSymbols.has(key) : isNonTrackableKeys(key)) {
      return res;
    }
    if (!isReadonly2) {
      track(target, "get", key);
    }
    if (isShallow2) {
      return res;
    }
    if (/* @__PURE__ */ isRef(res)) {
      const value = targetIsArray && isIntegerKey(key) ? res : res.value;
      return isReadonly2 && isObject$2(value) ? /* @__PURE__ */ readonly(value) : value;
    }
    if (isObject$2(res)) {
      return isReadonly2 ? /* @__PURE__ */ readonly(res) : /* @__PURE__ */ reactive(res);
    }
    return res;
  }
}
class MutableReactiveHandler extends BaseReactiveHandler {
  constructor(isShallow2 = false) {
    super(false, isShallow2);
  }
  set(target, key, value, receiver) {
    let oldValue = target[key];
    const isArrayWithIntegerKey = isArray$1(target) && isIntegerKey(key);
    if (!this._isShallow) {
      const isOldValueReadonly = /* @__PURE__ */ isReadonly(oldValue);
      if (!/* @__PURE__ */ isShallow(value) && !/* @__PURE__ */ isReadonly(value)) {
        oldValue = /* @__PURE__ */ toRaw(oldValue);
        value = /* @__PURE__ */ toRaw(value);
      }
      if (!isArrayWithIntegerKey && /* @__PURE__ */ isRef(oldValue) && !/* @__PURE__ */ isRef(value)) {
        if (isOldValueReadonly) {
          return true;
        } else {
          oldValue.value = value;
          return true;
        }
      }
    }
    const hadKey = isArrayWithIntegerKey ? Number(key) < target.length : hasOwn$1(target, key);
    const result = Reflect.set(
      target,
      key,
      value,
      /* @__PURE__ */ isRef(target) ? target : receiver
    );
    if (target === /* @__PURE__ */ toRaw(receiver)) {
      if (!hadKey) {
        trigger(target, "add", key, value);
      } else if (hasChanged(value, oldValue)) {
        trigger(target, "set", key, value);
      }
    }
    return result;
  }
  deleteProperty(target, key) {
    const hadKey = hasOwn$1(target, key);
    target[key];
    const result = Reflect.deleteProperty(target, key);
    if (result && hadKey) {
      trigger(target, "delete", key, void 0);
    }
    return result;
  }
  has(target, key) {
    const result = Reflect.has(target, key);
    if (!isSymbol(key) || !builtInSymbols.has(key)) {
      track(target, "has", key);
    }
    return result;
  }
  ownKeys(target) {
    track(
      target,
      "iterate",
      isArray$1(target) ? "length" : ITERATE_KEY
    );
    return Reflect.ownKeys(target);
  }
}
class ReadonlyReactiveHandler extends BaseReactiveHandler {
  constructor(isShallow2 = false) {
    super(true, isShallow2);
  }
  set(target, key) {
    return true;
  }
  deleteProperty(target, key) {
    return true;
  }
}
const mutableHandlers = /* @__PURE__ */ new MutableReactiveHandler();
const readonlyHandlers = /* @__PURE__ */ new ReadonlyReactiveHandler();
const shallowReactiveHandlers = /* @__PURE__ */ new MutableReactiveHandler(true);
const shallowReadonlyHandlers = /* @__PURE__ */ new ReadonlyReactiveHandler(true);
const toShallow = (value) => value;
const getProto = (v) => Reflect.getPrototypeOf(v);
function createIterableMethod(method, isReadonly2, isShallow2) {
  return function(...args) {
    const target = this["__v_raw"];
    const rawTarget = /* @__PURE__ */ toRaw(target);
    const targetIsMap = isMap(rawTarget);
    const isPair = method === "entries" || method === Symbol.iterator && targetIsMap;
    const isKeyOnly = method === "keys" && targetIsMap;
    const innerIterator = target[method](...args);
    const wrap = isShallow2 ? toShallow : isReadonly2 ? toReadonly : toReactive;
    !isReadonly2 && track(
      rawTarget,
      "iterate",
      isKeyOnly ? MAP_KEY_ITERATE_KEY : ITERATE_KEY
    );
    return extend(
      // inheriting all iterator properties
      Object.create(innerIterator),
      {
        // iterator protocol
        next() {
          const { value, done } = innerIterator.next();
          return done ? { value, done } : {
            value: isPair ? [wrap(value[0]), wrap(value[1])] : wrap(value),
            done
          };
        }
      }
    );
  };
}
function createReadonlyMethod(type) {
  return function(...args) {
    return type === "delete" ? false : type === "clear" ? void 0 : this;
  };
}
function createInstrumentations(readonly2, shallow) {
  const instrumentations = {
    get(key) {
      const target = this["__v_raw"];
      const rawTarget = /* @__PURE__ */ toRaw(target);
      const rawKey = /* @__PURE__ */ toRaw(key);
      if (!readonly2) {
        if (hasChanged(key, rawKey)) {
          track(rawTarget, "get", key);
        }
        track(rawTarget, "get", rawKey);
      }
      const { has } = getProto(rawTarget);
      const wrap = shallow ? toShallow : readonly2 ? toReadonly : toReactive;
      if (has.call(rawTarget, key)) {
        return wrap(target.get(key));
      } else if (has.call(rawTarget, rawKey)) {
        return wrap(target.get(rawKey));
      } else if (target !== rawTarget) {
        target.get(key);
      }
    },
    get size() {
      const target = this["__v_raw"];
      !readonly2 && track(/* @__PURE__ */ toRaw(target), "iterate", ITERATE_KEY);
      return target.size;
    },
    has(key) {
      const target = this["__v_raw"];
      const rawTarget = /* @__PURE__ */ toRaw(target);
      const rawKey = /* @__PURE__ */ toRaw(key);
      if (!readonly2) {
        if (hasChanged(key, rawKey)) {
          track(rawTarget, "has", key);
        }
        track(rawTarget, "has", rawKey);
      }
      return key === rawKey ? target.has(key) : target.has(key) || target.has(rawKey);
    },
    forEach(callback, thisArg) {
      const observed = this;
      const target = observed["__v_raw"];
      const rawTarget = /* @__PURE__ */ toRaw(target);
      const wrap = shallow ? toShallow : readonly2 ? toReadonly : toReactive;
      !readonly2 && track(rawTarget, "iterate", ITERATE_KEY);
      return target.forEach((value, key) => {
        return callback.call(thisArg, wrap(value), wrap(key), observed);
      });
    }
  };
  extend(
    instrumentations,
    readonly2 ? {
      add: createReadonlyMethod("add"),
      set: createReadonlyMethod("set"),
      delete: createReadonlyMethod("delete"),
      clear: createReadonlyMethod("clear")
    } : {
      add(value) {
        const target = /* @__PURE__ */ toRaw(this);
        const proto = getProto(target);
        const rawValue = /* @__PURE__ */ toRaw(value);
        const valueToAdd = !shallow && !/* @__PURE__ */ isShallow(value) && !/* @__PURE__ */ isReadonly(value) ? rawValue : value;
        const hadKey = proto.has.call(target, valueToAdd) || hasChanged(value, valueToAdd) && proto.has.call(target, value) || hasChanged(rawValue, valueToAdd) && proto.has.call(target, rawValue);
        if (!hadKey) {
          target.add(valueToAdd);
          trigger(target, "add", valueToAdd, valueToAdd);
        }
        return this;
      },
      set(key, value) {
        if (!shallow && !/* @__PURE__ */ isShallow(value) && !/* @__PURE__ */ isReadonly(value)) {
          value = /* @__PURE__ */ toRaw(value);
        }
        const target = /* @__PURE__ */ toRaw(this);
        const { has, get } = getProto(target);
        let hadKey = has.call(target, key);
        if (!hadKey) {
          key = /* @__PURE__ */ toRaw(key);
          hadKey = has.call(target, key);
        }
        const oldValue = get.call(target, key);
        target.set(key, value);
        if (!hadKey) {
          trigger(target, "add", key, value);
        } else if (hasChanged(value, oldValue)) {
          trigger(target, "set", key, value);
        }
        return this;
      },
      delete(key) {
        const target = /* @__PURE__ */ toRaw(this);
        const { has, get } = getProto(target);
        let hadKey = has.call(target, key);
        if (!hadKey) {
          key = /* @__PURE__ */ toRaw(key);
          hadKey = has.call(target, key);
        }
        get ? get.call(target, key) : void 0;
        const result = target.delete(key);
        if (hadKey) {
          trigger(target, "delete", key, void 0);
        }
        return result;
      },
      clear() {
        const target = /* @__PURE__ */ toRaw(this);
        const hadItems = target.size !== 0;
        const result = target.clear();
        if (hadItems) {
          trigger(
            target,
            "clear",
            void 0,
            void 0
          );
        }
        return result;
      }
    }
  );
  const iteratorMethods = [
    "keys",
    "values",
    "entries",
    Symbol.iterator
  ];
  iteratorMethods.forEach((method) => {
    instrumentations[method] = createIterableMethod(method, readonly2, shallow);
  });
  return instrumentations;
}
function createInstrumentationGetter(isReadonly2, shallow) {
  const instrumentations = createInstrumentations(isReadonly2, shallow);
  return (target, key, receiver) => {
    if (key === "__v_isReactive") {
      return !isReadonly2;
    } else if (key === "__v_isReadonly") {
      return isReadonly2;
    } else if (key === "__v_raw") {
      return target;
    }
    return Reflect.get(
      hasOwn$1(instrumentations, key) && key in target ? instrumentations : target,
      key,
      receiver
    );
  };
}
const mutableCollectionHandlers = {
  get: /* @__PURE__ */ createInstrumentationGetter(false, false)
};
const shallowCollectionHandlers = {
  get: /* @__PURE__ */ createInstrumentationGetter(false, true)
};
const readonlyCollectionHandlers = {
  get: /* @__PURE__ */ createInstrumentationGetter(true, false)
};
const shallowReadonlyCollectionHandlers = {
  get: /* @__PURE__ */ createInstrumentationGetter(true, true)
};
const reactiveMap = /* @__PURE__ */ new WeakMap();
const shallowReactiveMap = /* @__PURE__ */ new WeakMap();
const readonlyMap = /* @__PURE__ */ new WeakMap();
const shallowReadonlyMap = /* @__PURE__ */ new WeakMap();
function targetTypeMap(rawType) {
  switch (rawType) {
    case "Object":
    case "Array":
      return 1;
    case "Map":
    case "Set":
    case "WeakMap":
    case "WeakSet":
      return 2;
    default:
      return 0;
  }
}
// @__NO_SIDE_EFFECTS__
function reactive(target) {
  if (/* @__PURE__ */ isReadonly(target)) {
    return target;
  }
  return createReactiveObject(
    target,
    false,
    mutableHandlers,
    mutableCollectionHandlers,
    reactiveMap
  );
}
// @__NO_SIDE_EFFECTS__
function shallowReactive(target) {
  return createReactiveObject(
    target,
    false,
    shallowReactiveHandlers,
    shallowCollectionHandlers,
    shallowReactiveMap
  );
}
// @__NO_SIDE_EFFECTS__
function readonly(target) {
  return createReactiveObject(
    target,
    true,
    readonlyHandlers,
    readonlyCollectionHandlers,
    readonlyMap
  );
}
// @__NO_SIDE_EFFECTS__
function shallowReadonly(target) {
  return createReactiveObject(
    target,
    true,
    shallowReadonlyHandlers,
    shallowReadonlyCollectionHandlers,
    shallowReadonlyMap
  );
}
function createReactiveObject(target, isReadonly2, baseHandlers, collectionHandlers, proxyMap) {
  if (!isObject$2(target)) {
    return target;
  }
  if (target["__v_raw"] && !(isReadonly2 && target["__v_isReactive"])) {
    return target;
  }
  if (target["__v_skip"] || !Object.isExtensible(target)) {
    return target;
  }
  const existingProxy = proxyMap.get(target);
  if (existingProxy) {
    return existingProxy;
  }
  const targetType = targetTypeMap(toRawType(target));
  if (targetType === 0) {
    return target;
  }
  const proxy = new Proxy(
    target,
    targetType === 2 ? collectionHandlers : baseHandlers
  );
  proxyMap.set(target, proxy);
  return proxy;
}
// @__NO_SIDE_EFFECTS__
function isReactive(value) {
  if (/* @__PURE__ */ isReadonly(value)) {
    return /* @__PURE__ */ isReactive(value["__v_raw"]);
  }
  return !!(value && value["__v_isReactive"]);
}
// @__NO_SIDE_EFFECTS__
function isReadonly(value) {
  return !!(value && value["__v_isReadonly"]);
}
// @__NO_SIDE_EFFECTS__
function isShallow(value) {
  return !!(value && value["__v_isShallow"]);
}
// @__NO_SIDE_EFFECTS__
function isProxy(value) {
  return value ? !!value["__v_raw"] : false;
}
// @__NO_SIDE_EFFECTS__
function toRaw(observed) {
  const raw = observed && observed["__v_raw"];
  return raw ? /* @__PURE__ */ toRaw(raw) : observed;
}
function markRaw(value) {
  if (!hasOwn$1(value, "__v_skip") && Object.isExtensible(value)) {
    def(value, "__v_skip", true);
  }
  return value;
}
const toReactive = (value) => isObject$2(value) ? /* @__PURE__ */ reactive(value) : value;
const toReadonly = (value) => isObject$2(value) ? /* @__PURE__ */ readonly(value) : value;
// @__NO_SIDE_EFFECTS__
function isRef(r) {
  return r ? r["__v_isRef"] === true : false;
}
// @__NO_SIDE_EFFECTS__
function ref(value) {
  return createRef(value, false);
}
// @__NO_SIDE_EFFECTS__
function shallowRef(value) {
  return createRef(value, true);
}
function createRef(rawValue, shallow) {
  if (/* @__PURE__ */ isRef(rawValue)) {
    return rawValue;
  }
  return new RefImpl(rawValue, shallow);
}
class RefImpl {
  constructor(value, isShallow2) {
    this.dep = new Dep();
    this["__v_isRef"] = true;
    this["__v_isShallow"] = false;
    this._rawValue = isShallow2 ? value : /* @__PURE__ */ toRaw(value);
    this._value = isShallow2 ? value : toReactive(value);
    this["__v_isShallow"] = isShallow2;
  }
  get value() {
    {
      this.dep.track();
    }
    return this._value;
  }
  set value(newValue) {
    const oldValue = this._rawValue;
    const useDirectValue = this["__v_isShallow"] || /* @__PURE__ */ isShallow(newValue) || /* @__PURE__ */ isReadonly(newValue);
    newValue = useDirectValue ? newValue : /* @__PURE__ */ toRaw(newValue);
    if (hasChanged(newValue, oldValue)) {
      this._rawValue = newValue;
      this._value = useDirectValue ? newValue : toReactive(newValue);
      {
        this.dep.trigger();
      }
    }
  }
}
function unref(ref2) {
  return /* @__PURE__ */ isRef(ref2) ? ref2.value : ref2;
}
const shallowUnwrapHandlers = {
  get: (target, key, receiver) => key === "__v_raw" ? target : unref(Reflect.get(target, key, receiver)),
  set: (target, key, value, receiver) => {
    const oldValue = target[key];
    if (/* @__PURE__ */ isRef(oldValue) && !/* @__PURE__ */ isRef(value)) {
      oldValue.value = value;
      return true;
    } else {
      return Reflect.set(target, key, value, receiver);
    }
  }
};
function proxyRefs(objectWithRefs) {
  return /* @__PURE__ */ isReactive(objectWithRefs) ? objectWithRefs : new Proxy(objectWithRefs, shallowUnwrapHandlers);
}
// @__NO_SIDE_EFFECTS__
function toRefs(object) {
  const ret = isArray$1(object) ? new Array(object.length) : {};
  for (const key in object) {
    ret[key] = propertyToRef(object, key);
  }
  return ret;
}
class ObjectRefImpl {
  constructor(_object, key, _defaultValue) {
    this._object = _object;
    this._defaultValue = _defaultValue;
    this["__v_isRef"] = true;
    this._value = void 0;
    this._key = isSymbol(key) ? key : String(key);
    this._raw = /* @__PURE__ */ toRaw(_object);
    let shallow = true;
    let obj = _object;
    if (!isArray$1(_object) || isSymbol(this._key) || !isIntegerKey(this._key)) {
      do {
        shallow = !/* @__PURE__ */ isProxy(obj) || /* @__PURE__ */ isShallow(obj);
      } while (shallow && (obj = obj["__v_raw"]));
    }
    this._shallow = shallow;
  }
  get value() {
    let val = this._object[this._key];
    if (this._shallow) {
      val = unref(val);
    }
    return this._value = val === void 0 ? this._defaultValue : val;
  }
  set value(newVal) {
    if (this._shallow && /* @__PURE__ */ isRef(this._raw[this._key])) {
      const nestedRef = this._object[this._key];
      if (/* @__PURE__ */ isRef(nestedRef)) {
        nestedRef.value = newVal;
        return;
      }
    }
    this._object[this._key] = newVal;
  }
  get dep() {
    return getDepFromReactive(this._raw, this._key);
  }
}
class GetterRefImpl {
  constructor(_getter) {
    this._getter = _getter;
    this["__v_isRef"] = true;
    this["__v_isReadonly"] = true;
    this._value = void 0;
  }
  get value() {
    return this._value = this._getter();
  }
}
// @__NO_SIDE_EFFECTS__
function toRef(source, key, defaultValue) {
  if (/* @__PURE__ */ isRef(source)) {
    return source;
  } else if (isFunction$1(source)) {
    return new GetterRefImpl(source);
  } else if (isObject$2(source) && arguments.length > 1) {
    return propertyToRef(source, key, defaultValue);
  } else {
    return /* @__PURE__ */ ref(source);
  }
}
function propertyToRef(source, key, defaultValue) {
  return new ObjectRefImpl(source, key, defaultValue);
}
class ComputedRefImpl {
  constructor(fn, setter, isSSR) {
    this.fn = fn;
    this.setter = setter;
    this._value = void 0;
    this.dep = new Dep(this);
    this.__v_isRef = true;
    this.deps = void 0;
    this.depsTail = void 0;
    this.flags = 16;
    this.globalVersion = globalVersion - 1;
    this.next = void 0;
    this.effect = this;
    this["__v_isReadonly"] = !setter;
    this.isSSR = isSSR;
  }
  /**
   * @internal
   */
  notify() {
    this.flags |= 16;
    if (!(this.flags & 8) && // avoid infinite self recursion
    activeSub !== this) {
      batch(this, true);
      return true;
    }
  }
  get value() {
    const link = this.dep.track();
    refreshComputed(this);
    if (link) {
      link.version = this.dep.version;
    }
    return this._value;
  }
  set value(newValue) {
    if (this.setter) {
      this.setter(newValue);
    }
  }
}
// @__NO_SIDE_EFFECTS__
function computed$1(getterOrOptions, debugOptions, isSSR = false) {
  let getter;
  let setter;
  if (isFunction$1(getterOrOptions)) {
    getter = getterOrOptions;
  } else {
    getter = getterOrOptions.get;
    setter = getterOrOptions.set;
  }
  const cRef = new ComputedRefImpl(getter, setter, isSSR);
  return cRef;
}
const INITIAL_WATCHER_VALUE = {};
const cleanupMap = /* @__PURE__ */ new WeakMap();
let activeWatcher = void 0;
function onWatcherCleanup(cleanupFn, failSilently = false, owner = activeWatcher) {
  if (owner) {
    let cleanups = cleanupMap.get(owner);
    if (!cleanups) cleanupMap.set(owner, cleanups = []);
    cleanups.push(cleanupFn);
  }
}
function watch$1(source, cb, options = EMPTY_OBJ) {
  const { immediate, deep, once, scheduler, augmentJob, call } = options;
  const reactiveGetter = (source2) => {
    if (deep) return source2;
    if (/* @__PURE__ */ isShallow(source2) || deep === false || deep === 0)
      return traverse(source2, 1);
    return traverse(source2);
  };
  let effect2;
  let getter;
  let cleanup;
  let boundCleanup;
  let forceTrigger = false;
  let isMultiSource = false;
  if (/* @__PURE__ */ isRef(source)) {
    getter = () => source.value;
    forceTrigger = /* @__PURE__ */ isShallow(source);
  } else if (/* @__PURE__ */ isReactive(source)) {
    getter = () => reactiveGetter(source);
    forceTrigger = true;
  } else if (isArray$1(source)) {
    isMultiSource = true;
    forceTrigger = source.some((s) => /* @__PURE__ */ isReactive(s) || /* @__PURE__ */ isShallow(s));
    getter = () => source.map((s) => {
      if (/* @__PURE__ */ isRef(s)) {
        return s.value;
      } else if (/* @__PURE__ */ isReactive(s)) {
        return reactiveGetter(s);
      } else if (isFunction$1(s)) {
        return call ? call(s, 2) : s();
      } else ;
    });
  } else if (isFunction$1(source)) {
    if (cb) {
      getter = call ? () => call(source, 2) : source;
    } else {
      getter = () => {
        if (cleanup) {
          pauseTracking();
          try {
            cleanup();
          } finally {
            resetTracking();
          }
        }
        const currentEffect = activeWatcher;
        activeWatcher = effect2;
        try {
          return call ? call(source, 3, [boundCleanup]) : source(boundCleanup);
        } finally {
          activeWatcher = currentEffect;
        }
      };
    }
  } else {
    getter = NOOP;
  }
  if (cb && deep) {
    const baseGetter = getter;
    const depth = deep === true ? Infinity : deep;
    getter = () => traverse(baseGetter(), depth);
  }
  const scope = getCurrentScope();
  const watchHandle = () => {
    effect2.stop();
    if (scope && scope.active) {
      remove(scope.effects, effect2);
    }
  };
  if (once && cb) {
    const _cb = cb;
    cb = (...args) => {
      const res = _cb(...args);
      watchHandle();
      return res;
    };
  }
  let oldValue = isMultiSource ? new Array(source.length).fill(INITIAL_WATCHER_VALUE) : INITIAL_WATCHER_VALUE;
  const job = (immediateFirstRun) => {
    if (!(effect2.flags & 1) || !effect2.dirty && !immediateFirstRun) {
      return;
    }
    if (cb) {
      const newValue = effect2.run();
      if (immediateFirstRun || deep || forceTrigger || (isMultiSource ? newValue.some((v, i) => hasChanged(v, oldValue[i])) : hasChanged(newValue, oldValue))) {
        if (cleanup) {
          cleanup();
        }
        const currentWatcher = activeWatcher;
        activeWatcher = effect2;
        try {
          const args = [
            newValue,
            // pass undefined as the old value when it's changed for the first time
            oldValue === INITIAL_WATCHER_VALUE ? void 0 : isMultiSource && oldValue[0] === INITIAL_WATCHER_VALUE ? [] : oldValue,
            boundCleanup
          ];
          oldValue = newValue;
          call ? call(cb, 3, args) : (
            // @ts-expect-error
            cb(...args)
          );
        } finally {
          activeWatcher = currentWatcher;
        }
      }
    } else {
      effect2.run();
    }
  };
  if (augmentJob) {
    augmentJob(job);
  }
  effect2 = new ReactiveEffect(getter);
  effect2.scheduler = scheduler ? () => scheduler(job, false) : job;
  boundCleanup = (fn) => onWatcherCleanup(fn, false, effect2);
  cleanup = effect2.onStop = () => {
    const cleanups = cleanupMap.get(effect2);
    if (cleanups) {
      if (call) {
        call(cleanups, 4);
      } else {
        for (const cleanup2 of cleanups) cleanup2();
      }
      cleanupMap.delete(effect2);
    }
  };
  if (cb) {
    if (immediate) {
      job(true);
    } else {
      oldValue = effect2.run();
    }
  } else if (scheduler) {
    scheduler(job.bind(null, true), true);
  } else {
    effect2.run();
  }
  watchHandle.pause = effect2.pause.bind(effect2);
  watchHandle.resume = effect2.resume.bind(effect2);
  watchHandle.stop = watchHandle;
  return watchHandle;
}
function traverse(value, depth = Infinity, seen) {
  if (depth <= 0 || !isObject$2(value) || value["__v_skip"]) {
    return value;
  }
  seen = seen || /* @__PURE__ */ new Map();
  if ((seen.get(value) || 0) >= depth) {
    return value;
  }
  seen.set(value, depth);
  depth--;
  if (/* @__PURE__ */ isRef(value)) {
    traverse(value.value, depth, seen);
  } else if (isArray$1(value)) {
    for (let i = 0; i < value.length; i++) {
      traverse(value[i], depth, seen);
    }
  } else if (isSet(value) || isMap(value)) {
    value.forEach((v) => {
      traverse(v, depth, seen);
    });
  } else if (isPlainObject$2(value)) {
    for (const key in value) {
      traverse(value[key], depth, seen);
    }
    for (const key of Object.getOwnPropertySymbols(value)) {
      if (Object.prototype.propertyIsEnumerable.call(value, key)) {
        traverse(value[key], depth, seen);
      }
    }
  }
  return value;
}
/**
* @vue/runtime-core v3.5.38
* (c) 2018-present Yuxi (Evan) You and Vue contributors
* @license MIT
**/
const stack = [];
let isWarning = false;
function warn$1(msg, ...args) {
  if (isWarning) return;
  isWarning = true;
  pauseTracking();
  const instance = stack.length ? stack[stack.length - 1].component : null;
  const appWarnHandler = instance && instance.appContext.config.warnHandler;
  const trace = getComponentTrace();
  if (appWarnHandler) {
    callWithErrorHandling(
      appWarnHandler,
      instance,
      11,
      [
        // eslint-disable-next-line no-restricted-syntax
        msg + args.map((a) => {
          var _a, _b;
          return (_b = (_a = a.toString) == null ? void 0 : _a.call(a)) != null ? _b : JSON.stringify(a);
        }).join(""),
        instance && instance.proxy,
        trace.map(
          ({ vnode }) => `at <${formatComponentName(instance, vnode.type)}>`
        ).join("\n"),
        trace
      ]
    );
  } else {
    const warnArgs = [`[Vue warn]: ${msg}`, ...args];
    if (trace.length && // avoid spamming console during tests
    true) {
      warnArgs.push(`
`, ...formatTrace(trace));
    }
    console.warn(...warnArgs);
  }
  resetTracking();
  isWarning = false;
}
function getComponentTrace() {
  let currentVNode = stack[stack.length - 1];
  if (!currentVNode) {
    return [];
  }
  const normalizedStack = [];
  while (currentVNode) {
    const last = normalizedStack[0];
    if (last && last.vnode === currentVNode) {
      last.recurseCount++;
    } else {
      normalizedStack.push({
        vnode: currentVNode,
        recurseCount: 0
      });
    }
    const parentInstance = currentVNode.component && currentVNode.component.parent;
    currentVNode = parentInstance && parentInstance.vnode;
  }
  return normalizedStack;
}
function formatTrace(trace) {
  const logs = [];
  trace.forEach((entry, i) => {
    logs.push(...i === 0 ? [] : [`
`], ...formatTraceEntry(entry));
  });
  return logs;
}
function formatTraceEntry({ vnode, recurseCount }) {
  const postfix = recurseCount > 0 ? `... (${recurseCount} recursive calls)` : ``;
  const isRoot = vnode.component ? vnode.component.parent == null : false;
  const open = ` at <${formatComponentName(
    vnode.component,
    vnode.type,
    isRoot
  )}`;
  const close = `>` + postfix;
  return vnode.props ? [open, ...formatProps(vnode.props), close] : [open + close];
}
function formatProps(props) {
  const res = [];
  const keys = Object.keys(props);
  keys.slice(0, 3).forEach((key) => {
    res.push(...formatProp(key, props[key]));
  });
  if (keys.length > 3) {
    res.push(` ...`);
  }
  return res;
}
function formatProp(key, value, raw) {
  if (isString$2(value)) {
    value = JSON.stringify(value);
    return raw ? value : [`${key}=${value}`];
  } else if (typeof value === "number" || typeof value === "boolean" || value == null) {
    return raw ? value : [`${key}=${value}`];
  } else if (/* @__PURE__ */ isRef(value)) {
    value = formatProp(key, /* @__PURE__ */ toRaw(value.value), true);
    return raw ? value : [`${key}=Ref<`, value, `>`];
  } else if (isFunction$1(value)) {
    return [`${key}=fn${value.name ? `<${value.name}>` : ``}`];
  } else {
    value = /* @__PURE__ */ toRaw(value);
    return raw ? value : [`${key}=`, value];
  }
}
function callWithErrorHandling(fn, instance, type, args) {
  try {
    return args ? fn(...args) : fn();
  } catch (err) {
    handleError(err, instance, type);
  }
}
function callWithAsyncErrorHandling(fn, instance, type, args) {
  if (isFunction$1(fn)) {
    const res = callWithErrorHandling(fn, instance, type, args);
    if (res && isPromise$1(res)) {
      res.catch((err) => {
        handleError(err, instance, type);
      });
    }
    return res;
  }
  if (isArray$1(fn)) {
    const values = [];
    for (let i = 0; i < fn.length; i++) {
      values.push(callWithAsyncErrorHandling(fn[i], instance, type, args));
    }
    return values;
  }
}
function handleError(err, instance, type, throwInDev = true) {
  const contextVNode = instance ? instance.vnode : null;
  const { errorHandler, throwUnhandledErrorInProduction } = instance && instance.appContext.config || EMPTY_OBJ;
  if (instance) {
    let cur = instance.parent;
    const exposedInstance = instance.proxy;
    const errorInfo = `https://vuejs.org/error-reference/#runtime-${type}`;
    while (cur) {
      const errorCapturedHooks = cur.ec;
      if (errorCapturedHooks) {
        for (let i = 0; i < errorCapturedHooks.length; i++) {
          if (errorCapturedHooks[i](err, exposedInstance, errorInfo) === false) {
            return;
          }
        }
      }
      cur = cur.parent;
    }
    if (errorHandler) {
      pauseTracking();
      callWithErrorHandling(errorHandler, null, 10, [
        err,
        exposedInstance,
        errorInfo
      ]);
      resetTracking();
      return;
    }
  }
  logError(err, type, contextVNode, throwInDev, throwUnhandledErrorInProduction);
}
function logError(err, type, contextVNode, throwInDev = true, throwInProd = false) {
  if (throwInProd) {
    throw err;
  } else {
    console.error(err);
  }
}
const queue = [];
let flushIndex = -1;
const pendingPostFlushCbs = [];
let activePostFlushCbs = null;
let postFlushIndex = 0;
const resolvedPromise = /* @__PURE__ */ Promise.resolve();
let currentFlushPromise = null;
function nextTick(fn) {
  const p2 = currentFlushPromise || resolvedPromise;
  return fn ? p2.then(this ? fn.bind(this) : fn) : p2;
}
function findInsertionIndex(id) {
  let start = flushIndex + 1;
  let end = queue.length;
  while (start < end) {
    const middle = start + end >>> 1;
    const middleJob = queue[middle];
    const middleJobId = getId(middleJob);
    if (middleJobId < id || middleJobId === id && middleJob.flags & 2) {
      start = middle + 1;
    } else {
      end = middle;
    }
  }
  return start;
}
function queueJob(job) {
  if (!(job.flags & 1)) {
    const jobId = getId(job);
    const lastJob = queue[queue.length - 1];
    if (!lastJob || // fast path when the job id is larger than the tail
    !(job.flags & 2) && jobId >= getId(lastJob)) {
      queue.push(job);
    } else {
      queue.splice(findInsertionIndex(jobId), 0, job);
    }
    job.flags |= 1;
    queueFlush();
  }
}
function queueFlush() {
  if (!currentFlushPromise) {
    currentFlushPromise = resolvedPromise.then(flushJobs);
  }
}
function queuePostFlushCb(cb) {
  if (!isArray$1(cb)) {
    if (activePostFlushCbs && cb.id === -1) {
      activePostFlushCbs.splice(postFlushIndex + 1, 0, cb);
    } else if (!(cb.flags & 1)) {
      pendingPostFlushCbs.push(cb);
      cb.flags |= 1;
    }
  } else {
    pendingPostFlushCbs.push(...cb);
  }
  queueFlush();
}
function flushPreFlushCbs(instance, seen, i = flushIndex + 1) {
  for (; i < queue.length; i++) {
    const cb = queue[i];
    if (cb && cb.flags & 2) {
      if (instance && cb.id !== instance.uid) {
        continue;
      }
      queue.splice(i, 1);
      i--;
      if (cb.flags & 4) {
        cb.flags &= -2;
      }
      cb();
      if (!(cb.flags & 4)) {
        cb.flags &= -2;
      }
    }
  }
}
function flushPostFlushCbs(seen) {
  if (pendingPostFlushCbs.length) {
    const deduped = [...new Set(pendingPostFlushCbs)].sort(
      (a, b) => getId(a) - getId(b)
    );
    pendingPostFlushCbs.length = 0;
    if (activePostFlushCbs) {
      activePostFlushCbs.push(...deduped);
      return;
    }
    activePostFlushCbs = deduped;
    for (postFlushIndex = 0; postFlushIndex < activePostFlushCbs.length; postFlushIndex++) {
      const cb = activePostFlushCbs[postFlushIndex];
      if (cb.flags & 4) {
        cb.flags &= -2;
      }
      if (!(cb.flags & 8)) cb();
      cb.flags &= -2;
    }
    activePostFlushCbs = null;
    postFlushIndex = 0;
  }
}
const getId = (job) => job.id == null ? job.flags & 2 ? -1 : Infinity : job.id;
function flushJobs(seen) {
  try {
    for (flushIndex = 0; flushIndex < queue.length; flushIndex++) {
      const job = queue[flushIndex];
      if (job && !(job.flags & 8)) {
        if (false) ;
        if (job.flags & 4) {
          job.flags &= ~1;
        }
        callWithErrorHandling(
          job,
          job.i,
          job.i ? 15 : 14
        );
        if (!(job.flags & 4)) {
          job.flags &= ~1;
        }
      }
    }
  } finally {
    for (; flushIndex < queue.length; flushIndex++) {
      const job = queue[flushIndex];
      if (job) {
        job.flags &= -2;
      }
    }
    flushIndex = -1;
    queue.length = 0;
    flushPostFlushCbs();
    currentFlushPromise = null;
    if (queue.length || pendingPostFlushCbs.length) {
      flushJobs();
    }
  }
}
let currentRenderingInstance = null;
let currentScopeId = null;
function setCurrentRenderingInstance(instance) {
  const prev = currentRenderingInstance;
  currentRenderingInstance = instance;
  currentScopeId = instance && instance.type.__scopeId || null;
  return prev;
}
function withCtx(fn, ctx = currentRenderingInstance, isNonScopedSlot) {
  if (!ctx) return fn;
  if (fn._n) {
    return fn;
  }
  const renderFnWithContext = (...args) => {
    if (renderFnWithContext._d) {
      setBlockTracking(-1);
    }
    const prevInstance = setCurrentRenderingInstance(ctx);
    let res;
    try {
      res = fn(...args);
    } finally {
      setCurrentRenderingInstance(prevInstance);
      if (renderFnWithContext._d) {
        setBlockTracking(1);
      }
    }
    return res;
  };
  renderFnWithContext._n = true;
  renderFnWithContext._c = true;
  renderFnWithContext._d = true;
  return renderFnWithContext;
}
function withDirectives(vnode, directives) {
  if (currentRenderingInstance === null) {
    return vnode;
  }
  const instance = getComponentPublicInstance(currentRenderingInstance);
  const bindings = vnode.dirs || (vnode.dirs = []);
  for (let i = 0; i < directives.length; i++) {
    let [dir, value, arg, modifiers = EMPTY_OBJ] = directives[i];
    if (dir) {
      if (isFunction$1(dir)) {
        dir = {
          mounted: dir,
          updated: dir
        };
      }
      if (dir.deep) {
        traverse(value);
      }
      bindings.push({
        dir,
        instance,
        value,
        oldValue: void 0,
        arg,
        modifiers
      });
    }
  }
  return vnode;
}
function invokeDirectiveHook(vnode, prevVNode, instance, name) {
  const bindings = vnode.dirs;
  const oldBindings = prevVNode && prevVNode.dirs;
  for (let i = 0; i < bindings.length; i++) {
    const binding = bindings[i];
    if (oldBindings) {
      binding.oldValue = oldBindings[i].value;
    }
    let hook = binding.dir[name];
    if (hook) {
      pauseTracking();
      callWithAsyncErrorHandling(hook, instance, 8, [
        vnode.el,
        binding,
        vnode,
        prevVNode
      ]);
      resetTracking();
    }
  }
}
function provide(key, value) {
  if (currentInstance) {
    let provides = currentInstance.provides;
    const parentProvides = currentInstance.parent && currentInstance.parent.provides;
    if (parentProvides === provides) {
      provides = currentInstance.provides = Object.create(parentProvides);
    }
    provides[key] = value;
  }
}
function inject(key, defaultValue, treatDefaultAsFactory = false) {
  const instance = getCurrentInstance();
  if (instance || currentApp) {
    let provides = currentApp ? currentApp._context.provides : instance ? instance.parent == null || instance.ce ? instance.vnode.appContext && instance.vnode.appContext.provides : instance.parent.provides : void 0;
    if (provides && key in provides) {
      return provides[key];
    } else if (arguments.length > 1) {
      return treatDefaultAsFactory && isFunction$1(defaultValue) ? defaultValue.call(instance && instance.proxy) : defaultValue;
    } else ;
  }
}
function hasInjectionContext() {
  return !!(getCurrentInstance() || currentApp);
}
const ssrContextKey = /* @__PURE__ */ Symbol.for("v-scx");
const useSSRContext = () => {
  {
    const ctx = inject(ssrContextKey);
    return ctx;
  }
};
function watch(source, cb, options) {
  return doWatch(source, cb, options);
}
function doWatch(source, cb, options = EMPTY_OBJ) {
  const { immediate, deep, flush, once } = options;
  const baseWatchOptions = extend({}, options);
  const runsImmediately = cb && immediate || !cb && flush !== "post";
  let ssrCleanup;
  if (isInSSRComponentSetup) {
    if (flush === "sync") {
      const ctx = useSSRContext();
      ssrCleanup = ctx.__watcherHandles || (ctx.__watcherHandles = []);
    } else if (!runsImmediately) {
      const watchStopHandle = () => {
      };
      watchStopHandle.stop = NOOP;
      watchStopHandle.resume = NOOP;
      watchStopHandle.pause = NOOP;
      return watchStopHandle;
    }
  }
  const instance = currentInstance;
  baseWatchOptions.call = (fn, type, args) => callWithAsyncErrorHandling(fn, instance, type, args);
  let isPre = false;
  if (flush === "post") {
    baseWatchOptions.scheduler = (job) => {
      queuePostRenderEffect(job, instance && instance.suspense);
    };
  } else if (flush !== "sync") {
    isPre = true;
    baseWatchOptions.scheduler = (job, isFirstRun) => {
      if (isFirstRun) {
        job();
      } else {
        queueJob(job);
      }
    };
  }
  baseWatchOptions.augmentJob = (job) => {
    if (cb) {
      job.flags |= 4;
    }
    if (isPre) {
      job.flags |= 2;
      if (instance) {
        job.id = instance.uid;
        job.i = instance;
      }
    }
  };
  const watchHandle = watch$1(source, cb, baseWatchOptions);
  if (isInSSRComponentSetup) {
    if (ssrCleanup) {
      ssrCleanup.push(watchHandle);
    } else if (runsImmediately) {
      watchHandle();
    }
  }
  return watchHandle;
}
function instanceWatch(source, value, options) {
  const publicThis = this.proxy;
  const getter = isString$2(source) ? source.includes(".") ? createPathGetter(publicThis, source) : () => publicThis[source] : source.bind(publicThis, publicThis);
  let cb;
  if (isFunction$1(value)) {
    cb = value;
  } else {
    cb = value.handler;
    options = value;
  }
  const reset = setCurrentInstance(this);
  const res = doWatch(getter, cb.bind(publicThis), options);
  reset();
  return res;
}
function createPathGetter(ctx, path) {
  const segments = path.split(".");
  return () => {
    let cur = ctx;
    for (let i = 0; i < segments.length && cur; i++) {
      cur = cur[segments[i]];
    }
    return cur;
  };
}
const pendingMounts = /* @__PURE__ */ new WeakMap();
const TeleportEndKey = /* @__PURE__ */ Symbol("_vte");
const isTeleport = (type) => type.__isTeleport;
const isTeleportDisabled = (props) => props && (props.disabled || props.disabled === "");
const isTeleportDeferred = (props) => props && (props.defer || props.defer === "");
const isTargetSVG = (target) => typeof SVGElement !== "undefined" && target instanceof SVGElement;
const isTargetMathML = (target) => typeof MathMLElement === "function" && target instanceof MathMLElement;
const resolveTarget = (props, select) => {
  const targetSelector = props && props.to;
  if (isString$2(targetSelector)) {
    if (!select) {
      return null;
    } else {
      const target = select(targetSelector);
      return target;
    }
  } else {
    return targetSelector;
  }
};
const TeleportImpl = {
  name: "Teleport",
  __isTeleport: true,
  process(n1, n2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized, internals) {
    const {
      mc: mountChildren,
      pc: patchChildren,
      pbc: patchBlockChildren,
      o: { insert, querySelector, createText, createComment, parentNode }
    } = internals;
    const disabled = isTeleportDisabled(n2.props);
    let { dynamicChildren } = n2;
    const mount = (vnode, container2, anchor2) => {
      if (vnode.shapeFlag & 16) {
        mountChildren(
          vnode.children,
          container2,
          anchor2,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized
        );
      }
    };
    const mountToTarget = (vnode = n2) => {
      const disabled2 = isTeleportDisabled(vnode.props);
      const target = vnode.target = resolveTarget(vnode.props, querySelector);
      const targetAnchor = prepareAnchor(target, vnode, createText, insert);
      if (target) {
        if (namespace !== "svg" && isTargetSVG(target)) {
          namespace = "svg";
        } else if (namespace !== "mathml" && isTargetMathML(target)) {
          namespace = "mathml";
        }
        if (parentComponent && parentComponent.isCE) {
          (parentComponent.ce._teleportTargets || (parentComponent.ce._teleportTargets = /* @__PURE__ */ new Set())).add(target);
        }
        if (!disabled2) {
          mount(vnode, target, targetAnchor);
          updateCssVars(vnode, false);
        }
      }
    };
    const queuePendingMount = (vnode) => {
      const mountJob = () => {
        if (pendingMounts.get(vnode) !== mountJob) return;
        pendingMounts.delete(vnode);
        if (isTeleportDisabled(vnode.props)) {
          const mountContainer = parentNode(vnode.el) || container;
          mount(vnode, mountContainer, vnode.anchor);
          updateCssVars(vnode, true);
        }
        mountToTarget(vnode);
      };
      pendingMounts.set(vnode, mountJob);
      queuePostRenderEffect(mountJob, parentSuspense);
    };
    if (n1 == null) {
      const placeholder = n2.el = createText("");
      const mainAnchor = n2.anchor = createText("");
      insert(placeholder, container, anchor);
      insert(mainAnchor, container, anchor);
      if (isTeleportDeferred(n2.props) || parentSuspense && parentSuspense.pendingBranch) {
        queuePendingMount(n2);
        return;
      }
      if (disabled) {
        mount(n2, container, mainAnchor);
        updateCssVars(n2, true);
      }
      mountToTarget();
    } else {
      n2.el = n1.el;
      const mainAnchor = n2.anchor = n1.anchor;
      const pendingMount = pendingMounts.get(n1);
      if (pendingMount) {
        pendingMount.flags |= 8;
        pendingMounts.delete(n1);
        queuePendingMount(n2);
        return;
      }
      n2.targetStart = n1.targetStart;
      const target = n2.target = n1.target;
      const targetAnchor = n2.targetAnchor = n1.targetAnchor;
      const wasDisabled = isTeleportDisabled(n1.props);
      const currentContainer = wasDisabled ? container : target;
      const currentAnchor = wasDisabled ? mainAnchor : targetAnchor;
      if (namespace === "svg" || isTargetSVG(target)) {
        namespace = "svg";
      } else if (namespace === "mathml" || isTargetMathML(target)) {
        namespace = "mathml";
      }
      if (dynamicChildren) {
        patchBlockChildren(
          n1.dynamicChildren,
          dynamicChildren,
          currentContainer,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds
        );
        traverseStaticChildren(n1, n2, true);
      } else if (!optimized) {
        patchChildren(
          n1,
          n2,
          currentContainer,
          currentAnchor,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          false
        );
      }
      if (disabled) {
        if (!wasDisabled) {
          moveTeleport(
            n2,
            container,
            mainAnchor,
            internals,
            1
          );
        } else {
          if (n2.props && n1.props && n2.props.to !== n1.props.to) {
            n2.props.to = n1.props.to;
          }
        }
      } else {
        if ((n2.props && n2.props.to) !== (n1.props && n1.props.to)) {
          const nextTarget = n2.target = resolveTarget(
            n2.props,
            querySelector
          );
          if (nextTarget) {
            moveTeleport(
              n2,
              nextTarget,
              null,
              internals,
              0
            );
          }
        } else if (wasDisabled) {
          moveTeleport(
            n2,
            target,
            targetAnchor,
            internals,
            1
          );
        }
      }
      updateCssVars(n2, disabled);
    }
  },
  remove(vnode, parentComponent, parentSuspense, { um: unmount, o: { remove: hostRemove } }, doRemove) {
    const {
      shapeFlag,
      children,
      anchor,
      targetStart,
      targetAnchor,
      target,
      props
    } = vnode;
    const shouldRemove = doRemove || !isTeleportDisabled(props);
    const pendingMount = pendingMounts.get(vnode);
    if (pendingMount) {
      pendingMount.flags |= 8;
      pendingMounts.delete(vnode);
    }
    if (target) {
      hostRemove(targetStart);
      hostRemove(targetAnchor);
    }
    doRemove && hostRemove(anchor);
    if (!pendingMount && shapeFlag & 16) {
      for (let i = 0; i < children.length; i++) {
        const child = children[i];
        unmount(
          child,
          parentComponent,
          parentSuspense,
          shouldRemove,
          !!child.dynamicChildren
        );
      }
    }
  },
  move: moveTeleport,
  hydrate: hydrateTeleport
};
function moveTeleport(vnode, container, parentAnchor, { o: { insert }, m: move }, moveType = 2) {
  if (moveType === 0) {
    insert(vnode.targetAnchor, container, parentAnchor);
  }
  const { el, anchor, shapeFlag, children, props } = vnode;
  const isReorder = moveType === 2;
  if (isReorder) {
    insert(el, container, parentAnchor);
  }
  if (!pendingMounts.has(vnode) && (!isReorder || isTeleportDisabled(props))) {
    if (shapeFlag & 16) {
      for (let i = 0; i < children.length; i++) {
        move(
          children[i],
          container,
          parentAnchor,
          2
        );
      }
    }
  }
  if (isReorder) {
    insert(anchor, container, parentAnchor);
  }
}
function hydrateTeleport(node, vnode, parentComponent, parentSuspense, slotScopeIds, optimized, {
  o: { nextSibling, parentNode, querySelector, insert, createText }
}, hydrateChildren) {
  function hydrateAnchor(target2, targetNode) {
    let targetAnchor = targetNode;
    while (targetAnchor) {
      if (targetAnchor && targetAnchor.nodeType === 8) {
        if (targetAnchor.data === "teleport start anchor") {
          vnode.targetStart = targetAnchor;
        } else if (targetAnchor.data === "teleport anchor") {
          vnode.targetAnchor = targetAnchor;
          target2._lpa = vnode.targetAnchor && nextSibling(vnode.targetAnchor);
          break;
        }
      }
      targetAnchor = nextSibling(targetAnchor);
    }
  }
  function hydrateDisabledTeleport(node2, vnode2) {
    vnode2.anchor = hydrateChildren(
      nextSibling(node2),
      vnode2,
      parentNode(node2),
      parentComponent,
      parentSuspense,
      slotScopeIds,
      optimized
    );
  }
  const target = vnode.target = resolveTarget(
    vnode.props,
    querySelector
  );
  const disabled = isTeleportDisabled(vnode.props);
  if (target) {
    const targetNode = target._lpa || target.firstChild;
    if (vnode.shapeFlag & 16) {
      if (disabled) {
        hydrateDisabledTeleport(node, vnode);
        hydrateAnchor(target, targetNode);
        if (!vnode.targetAnchor) {
          prepareAnchor(
            target,
            vnode,
            createText,
            insert,
            // if target is the same as the main view, insert anchors before current node
            // to avoid hydrating mismatch
            parentNode(node) === target ? node : null
          );
        }
      } else {
        vnode.anchor = nextSibling(node);
        hydrateAnchor(target, targetNode);
        if (!vnode.targetAnchor) {
          prepareAnchor(target, vnode, createText, insert);
        }
        hydrateChildren(
          targetNode && nextSibling(targetNode),
          vnode,
          target,
          parentComponent,
          parentSuspense,
          slotScopeIds,
          optimized
        );
      }
    }
    updateCssVars(vnode, disabled);
  } else if (disabled) {
    if (vnode.shapeFlag & 16) {
      hydrateDisabledTeleport(node, vnode);
      vnode.targetStart = node;
      vnode.targetAnchor = nextSibling(node);
    }
  }
  return vnode.anchor && nextSibling(vnode.anchor);
}
const Teleport = TeleportImpl;
function updateCssVars(vnode, isDisabled) {
  const ctx = vnode.ctx;
  if (ctx && ctx.ut) {
    let node, anchor;
    if (isDisabled) {
      node = vnode.el;
      anchor = vnode.anchor;
    } else {
      node = vnode.targetStart;
      anchor = vnode.targetAnchor;
    }
    while (node && node !== anchor) {
      if (node.nodeType === 1) node.setAttribute("data-v-owner", ctx.uid);
      node = node.nextSibling;
    }
    ctx.ut();
  }
}
function prepareAnchor(target, vnode, createText, insert, anchor = null) {
  const targetStart = vnode.targetStart = createText("");
  const targetAnchor = vnode.targetAnchor = createText("");
  targetStart[TeleportEndKey] = targetAnchor;
  if (target) {
    insert(targetStart, target, anchor);
    insert(targetAnchor, target, anchor);
  }
  return targetAnchor;
}
const leaveCbKey = /* @__PURE__ */ Symbol("_leaveCb");
const enterCbKey$1 = /* @__PURE__ */ Symbol("_enterCb");
function useTransitionState() {
  const state = {
    isMounted: false,
    isLeaving: false,
    isUnmounting: false,
    leavingVNodes: /* @__PURE__ */ new Map()
  };
  onMounted(() => {
    state.isMounted = true;
  });
  onBeforeUnmount(() => {
    state.isUnmounting = true;
  });
  return state;
}
const TransitionHookValidator = [Function, Array];
const BaseTransitionPropsValidators = {
  mode: String,
  appear: Boolean,
  persisted: Boolean,
  // enter
  onBeforeEnter: TransitionHookValidator,
  onEnter: TransitionHookValidator,
  onAfterEnter: TransitionHookValidator,
  onEnterCancelled: TransitionHookValidator,
  // leave
  onBeforeLeave: TransitionHookValidator,
  onLeave: TransitionHookValidator,
  onAfterLeave: TransitionHookValidator,
  onLeaveCancelled: TransitionHookValidator,
  // appear
  onBeforeAppear: TransitionHookValidator,
  onAppear: TransitionHookValidator,
  onAfterAppear: TransitionHookValidator,
  onAppearCancelled: TransitionHookValidator
};
const recursiveGetSubtree = (instance) => {
  const subTree = instance.subTree;
  return subTree.component ? recursiveGetSubtree(subTree.component) : subTree;
};
const BaseTransitionImpl = {
  name: `BaseTransition`,
  props: BaseTransitionPropsValidators,
  setup(props, { slots }) {
    const instance = getCurrentInstance();
    const state = useTransitionState();
    return () => {
      const children = slots.default && getTransitionRawChildren(slots.default(), true);
      const child = children && children.length ? findNonCommentChild(children) : (
        // Keep explicit default-slot conditionals on the same transition path
        // as regular v-if branches, which render a comment placeholder.
        instance.subTree ? createCommentVNode() : void 0
      );
      if (!child) {
        return;
      }
      const rawProps = /* @__PURE__ */ toRaw(props);
      const { mode } = rawProps;
      if (state.isLeaving) {
        return emptyPlaceholder(child);
      }
      const innerChild = getInnerChild$1(child);
      if (!innerChild) {
        return emptyPlaceholder(child);
      }
      let enterHooks = resolveTransitionHooks(
        innerChild,
        rawProps,
        state,
        instance,
        // #11061, ensure enterHooks is fresh after clone
        (hooks) => enterHooks = hooks
      );
      if (innerChild.type !== Comment) {
        setTransitionHooks(innerChild, enterHooks);
      }
      let oldInnerChild = instance.subTree && getInnerChild$1(instance.subTree);
      if (oldInnerChild && oldInnerChild.type !== Comment && !isSameVNodeType(oldInnerChild, innerChild) && recursiveGetSubtree(instance).type !== Comment) {
        let leavingHooks = resolveTransitionHooks(
          oldInnerChild,
          rawProps,
          state,
          instance
        );
        setTransitionHooks(oldInnerChild, leavingHooks);
        if (mode === "out-in" && innerChild.type !== Comment) {
          state.isLeaving = true;
          leavingHooks.afterLeave = () => {
            state.isLeaving = false;
            if (!(instance.job.flags & 8)) {
              instance.update();
            }
            delete leavingHooks.afterLeave;
            oldInnerChild = void 0;
          };
          return emptyPlaceholder(child);
        } else if (mode === "in-out" && innerChild.type !== Comment) {
          leavingHooks.delayLeave = (el, earlyRemove, delayedLeave) => {
            const leavingVNodesCache = getLeavingNodesForType(
              state,
              oldInnerChild
            );
            leavingVNodesCache[String(oldInnerChild.key)] = oldInnerChild;
            el[leaveCbKey] = () => {
              earlyRemove();
              el[leaveCbKey] = void 0;
              delete enterHooks.delayedLeave;
              oldInnerChild = void 0;
            };
            enterHooks.delayedLeave = () => {
              delayedLeave();
              delete enterHooks.delayedLeave;
              oldInnerChild = void 0;
            };
          };
        } else {
          oldInnerChild = void 0;
        }
      } else if (oldInnerChild) {
        oldInnerChild = void 0;
      }
      return child;
    };
  }
};
function findNonCommentChild(children) {
  let child = children[0];
  if (children.length > 1) {
    for (const c of children) {
      if (c.type !== Comment) {
        child = c;
        break;
      }
    }
  }
  return child;
}
const BaseTransition = BaseTransitionImpl;
function getLeavingNodesForType(state, vnode) {
  const { leavingVNodes } = state;
  let leavingVNodesCache = leavingVNodes.get(vnode.type);
  if (!leavingVNodesCache) {
    leavingVNodesCache = /* @__PURE__ */ Object.create(null);
    leavingVNodes.set(vnode.type, leavingVNodesCache);
  }
  return leavingVNodesCache;
}
function resolveTransitionHooks(vnode, props, state, instance, postClone) {
  const {
    appear,
    mode,
    persisted = false,
    onBeforeEnter,
    onEnter,
    onAfterEnter,
    onEnterCancelled,
    onBeforeLeave,
    onLeave,
    onAfterLeave,
    onLeaveCancelled,
    onBeforeAppear,
    onAppear,
    onAfterAppear,
    onAppearCancelled
  } = props;
  const key = String(vnode.key);
  const leavingVNodesCache = getLeavingNodesForType(state, vnode);
  const callHook2 = (hook, args) => {
    hook && callWithAsyncErrorHandling(
      hook,
      instance,
      9,
      args
    );
  };
  const callAsyncHook = (hook, args) => {
    const done = args[1];
    callHook2(hook, args);
    if (isArray$1(hook)) {
      if (hook.every((hook2) => hook2.length <= 1)) done();
    } else if (hook.length <= 1) {
      done();
    }
  };
  const hooks = {
    mode,
    persisted,
    beforeEnter(el) {
      let hook = onBeforeEnter;
      if (!state.isMounted) {
        if (appear) {
          hook = onBeforeAppear || onBeforeEnter;
        } else {
          return;
        }
      }
      if (el[leaveCbKey]) {
        el[leaveCbKey](
          true
          /* cancelled */
        );
      }
      const leavingVNode = leavingVNodesCache[key];
      if (leavingVNode && isSameVNodeType(vnode, leavingVNode) && leavingVNode.el[leaveCbKey]) {
        leavingVNode.el[leaveCbKey]();
      }
      callHook2(hook, [el]);
    },
    enter(el) {
      if (leavingVNodesCache[key] === vnode) return;
      let hook = onEnter;
      let afterHook = onAfterEnter;
      let cancelHook = onEnterCancelled;
      if (!state.isMounted) {
        if (appear) {
          hook = onAppear || onEnter;
          afterHook = onAfterAppear || onAfterEnter;
          cancelHook = onAppearCancelled || onEnterCancelled;
        } else {
          return;
        }
      }
      let called = false;
      el[enterCbKey$1] = (cancelled) => {
        if (called) return;
        called = true;
        if (cancelled) {
          callHook2(cancelHook, [el]);
        } else {
          callHook2(afterHook, [el]);
        }
        if (hooks.delayedLeave) {
          hooks.delayedLeave();
        }
        el[enterCbKey$1] = void 0;
      };
      const done = el[enterCbKey$1].bind(null, false);
      if (hook) {
        callAsyncHook(hook, [el, done]);
      } else {
        done();
      }
    },
    leave(el, remove2) {
      const key2 = String(vnode.key);
      if (el[enterCbKey$1]) {
        el[enterCbKey$1](
          true
          /* cancelled */
        );
      }
      if (state.isUnmounting) {
        return remove2();
      }
      callHook2(onBeforeLeave, [el]);
      let called = false;
      el[leaveCbKey] = (cancelled) => {
        if (called) return;
        called = true;
        remove2();
        if (cancelled) {
          callHook2(onLeaveCancelled, [el]);
        } else {
          callHook2(onAfterLeave, [el]);
        }
        el[leaveCbKey] = void 0;
        if (leavingVNodesCache[key2] === vnode) {
          delete leavingVNodesCache[key2];
        }
      };
      const done = el[leaveCbKey].bind(null, false);
      leavingVNodesCache[key2] = vnode;
      if (onLeave) {
        callAsyncHook(onLeave, [el, done]);
      } else {
        done();
      }
    },
    clone(vnode2) {
      const hooks2 = resolveTransitionHooks(
        vnode2,
        props,
        state,
        instance,
        postClone
      );
      if (postClone) postClone(hooks2);
      return hooks2;
    }
  };
  return hooks;
}
function emptyPlaceholder(vnode) {
  if (isKeepAlive(vnode)) {
    vnode = cloneVNode(vnode);
    vnode.children = null;
    return vnode;
  }
}
function getInnerChild$1(vnode) {
  if (!isKeepAlive(vnode)) {
    if (isTeleport(vnode.type) && vnode.children) {
      return findNonCommentChild(vnode.children);
    }
    return vnode;
  }
  if (vnode.component) {
    return vnode.component.subTree;
  }
  const { shapeFlag, children } = vnode;
  if (children) {
    if (shapeFlag & 16) {
      return children[0];
    }
    if (shapeFlag & 32 && isFunction$1(children.default)) {
      return children.default();
    }
  }
}
function setTransitionHooks(vnode, hooks) {
  if (vnode.shapeFlag & 6 && vnode.component) {
    vnode.transition = hooks;
    setTransitionHooks(vnode.component.subTree, hooks);
  } else if (vnode.shapeFlag & 128) {
    vnode.ssContent.transition = hooks.clone(vnode.ssContent);
    vnode.ssFallback.transition = hooks.clone(vnode.ssFallback);
  } else {
    vnode.transition = hooks;
  }
}
function getTransitionRawChildren(children, keepComment = false, parentKey) {
  let ret = [];
  let keyedFragmentCount = 0;
  for (let i = 0; i < children.length; i++) {
    let child = children[i];
    const key = parentKey == null ? child.key : String(parentKey) + String(child.key != null ? child.key : i);
    if (child.type === Fragment) {
      if (child.patchFlag & 128) keyedFragmentCount++;
      ret = ret.concat(
        getTransitionRawChildren(child.children, keepComment, key)
      );
    } else if (keepComment || child.type !== Comment) {
      ret.push(key != null ? cloneVNode(child, { key }) : child);
    }
  }
  if (keyedFragmentCount > 1) {
    for (let i = 0; i < ret.length; i++) {
      ret[i].patchFlag = -2;
    }
  }
  return ret;
}
// @__NO_SIDE_EFFECTS__
function defineComponent(options, extraOptions) {
  return isFunction$1(options) ? (
    // #8236: extend call and options.name access are considered side-effects
    // by Rollup, so we have to wrap it in a pure-annotated IIFE.
    /* @__PURE__ */ (() => extend({ name: options.name }, extraOptions, { setup: options }))()
  ) : options;
}
function markAsyncBoundary(instance) {
  instance.ids = [instance.ids[0] + instance.ids[2]++ + "-", 0, 0];
}
function isTemplateRefKey(refs, key) {
  let desc;
  return !!((desc = Object.getOwnPropertyDescriptor(refs, key)) && !desc.configurable);
}
const pendingSetRefMap = /* @__PURE__ */ new WeakMap();
function setRef(rawRef, oldRawRef, parentSuspense, vnode, isUnmount = false) {
  if (isArray$1(rawRef)) {
    rawRef.forEach(
      (r, i) => setRef(
        r,
        oldRawRef && (isArray$1(oldRawRef) ? oldRawRef[i] : oldRawRef),
        parentSuspense,
        vnode,
        isUnmount
      )
    );
    return;
  }
  if (isAsyncWrapper(vnode) && !isUnmount) {
    if (vnode.shapeFlag & 512 && vnode.type.__asyncResolved && vnode.component.subTree.component) {
      setRef(rawRef, oldRawRef, parentSuspense, vnode.component.subTree);
    }
    return;
  }
  const refValue = vnode.shapeFlag & 4 ? getComponentPublicInstance(vnode.component) : vnode.el;
  const value = isUnmount ? null : refValue;
  const { i: owner, r: ref3 } = rawRef;
  const oldRef = oldRawRef && oldRawRef.r;
  const refs = owner.refs === EMPTY_OBJ ? owner.refs = {} : owner.refs;
  const setupState = owner.setupState;
  const rawSetupState = /* @__PURE__ */ toRaw(setupState);
  const canSetSetupRef = setupState === EMPTY_OBJ ? NO : (key) => {
    if (isTemplateRefKey(refs, key)) {
      return false;
    }
    return hasOwn$1(rawSetupState, key);
  };
  const canSetRef = (ref22, key) => {
    if (key && isTemplateRefKey(refs, key)) {
      return false;
    }
    return true;
  };
  if (oldRef != null && oldRef !== ref3) {
    invalidatePendingSetRef(oldRawRef);
    if (isString$2(oldRef)) {
      refs[oldRef] = null;
      if (canSetSetupRef(oldRef)) {
        setupState[oldRef] = null;
      }
    } else if (/* @__PURE__ */ isRef(oldRef)) {
      const oldRawRefAtom = oldRawRef;
      if (canSetRef(oldRef, oldRawRefAtom.k)) {
        oldRef.value = null;
      }
      if (oldRawRefAtom.k) refs[oldRawRefAtom.k] = null;
    }
  }
  if (isFunction$1(ref3)) {
    callWithErrorHandling(ref3, owner, 12, [value, refs]);
  } else {
    const _isString = isString$2(ref3);
    const _isRef = /* @__PURE__ */ isRef(ref3);
    if (_isString || _isRef) {
      const doSet = () => {
        if (rawRef.f) {
          const existing = _isString ? canSetSetupRef(ref3) ? setupState[ref3] : refs[ref3] : canSetRef() || !rawRef.k ? ref3.value : refs[rawRef.k];
          if (isUnmount) {
            isArray$1(existing) && remove(existing, refValue);
          } else {
            if (!isArray$1(existing)) {
              if (_isString) {
                refs[ref3] = [refValue];
                if (canSetSetupRef(ref3)) {
                  setupState[ref3] = refs[ref3];
                }
              } else {
                const newVal = [refValue];
                if (canSetRef(ref3, rawRef.k)) {
                  ref3.value = newVal;
                }
                if (rawRef.k) refs[rawRef.k] = newVal;
              }
            } else if (!existing.includes(refValue)) {
              existing.push(refValue);
            }
          }
        } else if (_isString) {
          refs[ref3] = value;
          if (canSetSetupRef(ref3)) {
            setupState[ref3] = value;
          }
        } else if (_isRef) {
          if (canSetRef(ref3, rawRef.k)) {
            ref3.value = value;
          }
          if (rawRef.k) refs[rawRef.k] = value;
        } else ;
      };
      if (value) {
        const job = () => {
          doSet();
          pendingSetRefMap.delete(rawRef);
        };
        job.id = -1;
        pendingSetRefMap.set(rawRef, job);
        queuePostRenderEffect(job, parentSuspense);
      } else {
        invalidatePendingSetRef(rawRef);
        doSet();
      }
    }
  }
}
function invalidatePendingSetRef(rawRef) {
  const pendingSetRef = pendingSetRefMap.get(rawRef);
  if (pendingSetRef) {
    pendingSetRef.flags |= 8;
    pendingSetRefMap.delete(rawRef);
  }
}
getGlobalThis$1().requestIdleCallback || ((cb) => setTimeout(cb, 1));
getGlobalThis$1().cancelIdleCallback || ((id) => clearTimeout(id));
const isAsyncWrapper = (i) => !!i.type.__asyncLoader;
const isKeepAlive = (vnode) => vnode.type.__isKeepAlive;
function onActivated(hook, target) {
  registerKeepAliveHook(hook, "a", target);
}
function onDeactivated(hook, target) {
  registerKeepAliveHook(hook, "da", target);
}
function registerKeepAliveHook(hook, type, target = currentInstance) {
  const wrappedHook = hook.__wdc || (hook.__wdc = () => {
    let current = target;
    while (current) {
      if (current.isDeactivated) {
        return;
      }
      current = current.parent;
    }
    return hook();
  });
  injectHook(type, wrappedHook, target);
  if (target) {
    let current = target.parent;
    while (current && current.parent) {
      if (isKeepAlive(current.parent.vnode)) {
        injectToKeepAliveRoot(wrappedHook, type, target, current);
      }
      current = current.parent;
    }
  }
}
function injectToKeepAliveRoot(hook, type, target, keepAliveRoot) {
  const injected = injectHook(
    type,
    hook,
    keepAliveRoot,
    true
    /* prepend */
  );
  onUnmounted(() => {
    remove(keepAliveRoot[type], injected);
  }, target);
}
function injectHook(type, hook, target = currentInstance, prepend = false) {
  if (target) {
    const hooks = target[type] || (target[type] = []);
    const wrappedHook = hook.__weh || (hook.__weh = (...args) => {
      pauseTracking();
      const reset = setCurrentInstance(target);
      const res = callWithAsyncErrorHandling(hook, target, type, args);
      reset();
      resetTracking();
      return res;
    });
    if (prepend) {
      hooks.unshift(wrappedHook);
    } else {
      hooks.push(wrappedHook);
    }
    return wrappedHook;
  }
}
const createHook = (lifecycle) => (hook, target = currentInstance) => {
  if (!isInSSRComponentSetup || lifecycle === "sp") {
    injectHook(lifecycle, (...args) => hook(...args), target);
  }
};
const onBeforeMount = createHook("bm");
const onMounted = createHook("m");
const onBeforeUpdate = createHook(
  "bu"
);
const onUpdated = createHook("u");
const onBeforeUnmount = createHook(
  "bum"
);
const onUnmounted = createHook("um");
const onServerPrefetch = createHook(
  "sp"
);
const onRenderTriggered = createHook("rtg");
const onRenderTracked = createHook("rtc");
function onErrorCaptured(hook, target = currentInstance) {
  injectHook("ec", hook, target);
}
const COMPONENTS = "components";
function resolveComponent(name, maybeSelfReference) {
  return resolveAsset(COMPONENTS, name, true, maybeSelfReference) || name;
}
const NULL_DYNAMIC_COMPONENT = /* @__PURE__ */ Symbol.for("v-ndc");
function resolveAsset(type, name, warnMissing = true, maybeSelfReference = false) {
  const instance = currentRenderingInstance || currentInstance;
  if (instance) {
    const Component = instance.type;
    {
      const selfName = getComponentName(
        Component,
        false
      );
      if (selfName && (selfName === name || selfName === camelize(name) || selfName === capitalize$1(camelize(name)))) {
        return Component;
      }
    }
    const res = (
      // local registration
      // check instance[type] first which is resolved for options API
      resolve(instance[type] || Component[type], name) || // global registration
      resolve(instance.appContext[type], name)
    );
    if (!res && maybeSelfReference) {
      return Component;
    }
    return res;
  }
}
function resolve(registry, name) {
  return registry && (registry[name] || registry[camelize(name)] || registry[capitalize$1(camelize(name))]);
}
function renderList(source, renderItem, cache2, index) {
  let ret;
  const cached2 = cache2;
  const sourceIsArray = isArray$1(source);
  if (sourceIsArray || isString$2(source)) {
    const sourceIsReactiveArray = sourceIsArray && /* @__PURE__ */ isReactive(source);
    let needsWrap = false;
    let isReadonlySource = false;
    if (sourceIsReactiveArray) {
      needsWrap = !/* @__PURE__ */ isShallow(source);
      isReadonlySource = /* @__PURE__ */ isReadonly(source);
      source = shallowReadArray(source);
    }
    ret = new Array(source.length);
    for (let i = 0, l = source.length; i < l; i++) {
      ret[i] = renderItem(
        needsWrap ? isReadonlySource ? toReadonly(toReactive(source[i])) : toReactive(source[i]) : source[i],
        i,
        void 0,
        cached2
      );
    }
  } else if (typeof source === "number") {
    {
      ret = new Array(source);
      for (let i = 0; i < source; i++) {
        ret[i] = renderItem(i + 1, i, void 0, cached2);
      }
    }
  } else if (isObject$2(source)) {
    if (source[Symbol.iterator]) {
      ret = Array.from(
        source,
        (item, i) => renderItem(item, i, void 0, cached2)
      );
    } else {
      const keys = Object.keys(source);
      ret = new Array(keys.length);
      for (let i = 0, l = keys.length; i < l; i++) {
        const key = keys[i];
        ret[i] = renderItem(source[key], key, i, cached2);
      }
    }
  } else {
    ret = [];
  }
  return ret;
}
function renderSlot(slots, name, props = {}, fallback, noSlotted) {
  if (currentRenderingInstance.ce || currentRenderingInstance.parent && isAsyncWrapper(currentRenderingInstance.parent) && currentRenderingInstance.parent.ce) {
    const hasProps = Object.keys(props).length > 0;
    if (name !== "default") props.name = name;
    return openBlock(), createBlock(
      Fragment,
      null,
      [createVNode("slot", props, fallback && fallback())],
      hasProps ? -2 : 64
    );
  }
  let slot = slots[name];
  if (slot && slot._c) {
    slot._d = false;
  }
  openBlock();
  const validSlotContent = slot && ensureValidVNode(slot(props));
  const slotKey = props.key || // slot content array of a dynamic conditional slot may have a branch
  // key attached in the `createSlots` helper, respect that
  validSlotContent && validSlotContent.key;
  const rendered = createBlock(
    Fragment,
    {
      key: (slotKey && !isSymbol(slotKey) ? slotKey : `_${name}`) + // #7256 force differentiate fallback content from actual content
      (!validSlotContent && fallback ? "_fb" : "")
    },
    validSlotContent || (fallback ? fallback() : []),
    validSlotContent && slots._ === 1 ? 64 : -2
  );
  if (slot && slot._c) {
    slot._d = true;
  }
  return rendered;
}
function ensureValidVNode(vnodes) {
  return vnodes.some((child) => {
    if (!isVNode$1(child)) return true;
    if (child.type === Comment) return false;
    if (child.type === Fragment && !ensureValidVNode(child.children))
      return false;
    return true;
  }) ? vnodes : null;
}
const getPublicInstance = (i) => {
  if (!i) return null;
  if (isStatefulComponent(i)) return getComponentPublicInstance(i);
  return getPublicInstance(i.parent);
};
const publicPropertiesMap = (
  // Move PURE marker to new line to workaround compiler discarding it
  // due to type annotation
  /* @__PURE__ */ extend(/* @__PURE__ */ Object.create(null), {
    $: (i) => i,
    $el: (i) => i.vnode.el,
    $data: (i) => i.data,
    $props: (i) => i.props,
    $attrs: (i) => i.attrs,
    $slots: (i) => i.slots,
    $refs: (i) => i.refs,
    $parent: (i) => getPublicInstance(i.parent),
    $root: (i) => getPublicInstance(i.root),
    $host: (i) => i.ce,
    $emit: (i) => i.emit,
    $options: (i) => resolveMergedOptions(i),
    $forceUpdate: (i) => i.f || (i.f = () => {
      queueJob(i.update);
    }),
    $nextTick: (i) => i.n || (i.n = nextTick.bind(i.proxy)),
    $watch: (i) => instanceWatch.bind(i)
  })
);
const hasSetupBinding = (state, key) => state !== EMPTY_OBJ && !state.__isScriptSetup && hasOwn$1(state, key);
const PublicInstanceProxyHandlers = {
  get({ _: instance }, key) {
    if (key === "__v_skip") {
      return true;
    }
    const { ctx, setupState, data, props, accessCache, type, appContext } = instance;
    if (key[0] !== "$") {
      const n = accessCache[key];
      if (n !== void 0) {
        switch (n) {
          case 1:
            return setupState[key];
          case 2:
            return data[key];
          case 4:
            return ctx[key];
          case 3:
            return props[key];
        }
      } else if (hasSetupBinding(setupState, key)) {
        accessCache[key] = 1;
        return setupState[key];
      } else if (data !== EMPTY_OBJ && hasOwn$1(data, key)) {
        accessCache[key] = 2;
        return data[key];
      } else if (hasOwn$1(props, key)) {
        accessCache[key] = 3;
        return props[key];
      } else if (ctx !== EMPTY_OBJ && hasOwn$1(ctx, key)) {
        accessCache[key] = 4;
        return ctx[key];
      } else if (shouldCacheAccess) {
        accessCache[key] = 0;
      }
    }
    const publicGetter = publicPropertiesMap[key];
    let cssModule, globalProperties;
    if (publicGetter) {
      if (key === "$attrs") {
        track(instance.attrs, "get", "");
      }
      return publicGetter(instance);
    } else if (
      // css module (injected by vue-loader)
      (cssModule = type.__cssModules) && (cssModule = cssModule[key])
    ) {
      return cssModule;
    } else if (ctx !== EMPTY_OBJ && hasOwn$1(ctx, key)) {
      accessCache[key] = 4;
      return ctx[key];
    } else if (
      // global properties
      globalProperties = appContext.config.globalProperties, hasOwn$1(globalProperties, key)
    ) {
      {
        return globalProperties[key];
      }
    } else ;
  },
  set({ _: instance }, key, value) {
    const { data, setupState, ctx } = instance;
    if (hasSetupBinding(setupState, key)) {
      setupState[key] = value;
      return true;
    } else if (data !== EMPTY_OBJ && hasOwn$1(data, key)) {
      data[key] = value;
      return true;
    } else if (hasOwn$1(instance.props, key)) {
      return false;
    }
    if (key[0] === "$" && key.slice(1) in instance) {
      return false;
    } else {
      {
        ctx[key] = value;
      }
    }
    return true;
  },
  has({
    _: { data, setupState, accessCache, ctx, appContext, props, type }
  }, key) {
    let cssModules;
    return !!(accessCache[key] || data !== EMPTY_OBJ && key[0] !== "$" && hasOwn$1(data, key) || hasSetupBinding(setupState, key) || hasOwn$1(props, key) || hasOwn$1(ctx, key) || hasOwn$1(publicPropertiesMap, key) || hasOwn$1(appContext.config.globalProperties, key) || (cssModules = type.__cssModules) && cssModules[key]);
  },
  defineProperty(target, key, descriptor) {
    if (descriptor.get != null) {
      target._.accessCache[key] = 0;
    } else if (hasOwn$1(descriptor, "value")) {
      this.set(target, key, descriptor.value, null);
    }
    return Reflect.defineProperty(target, key, descriptor);
  }
};
function normalizePropsOrEmits(props) {
  return isArray$1(props) ? props.reduce(
    (normalized, p2) => (normalized[p2] = null, normalized),
    {}
  ) : props;
}
let shouldCacheAccess = true;
function applyOptions(instance) {
  const options = resolveMergedOptions(instance);
  const publicThis = instance.proxy;
  const ctx = instance.ctx;
  shouldCacheAccess = false;
  if (options.beforeCreate) {
    callHook$1(options.beforeCreate, instance, "bc");
  }
  const {
    // state
    data: dataOptions,
    computed: computedOptions,
    methods,
    watch: watchOptions,
    provide: provideOptions,
    inject: injectOptions,
    // lifecycle
    created,
    beforeMount,
    mounted,
    beforeUpdate,
    updated,
    activated,
    deactivated,
    beforeDestroy,
    beforeUnmount,
    destroyed,
    unmounted,
    render: render2,
    renderTracked,
    renderTriggered,
    errorCaptured,
    serverPrefetch,
    // public API
    expose,
    inheritAttrs,
    // assets
    components,
    directives,
    filters
  } = options;
  const checkDuplicateProperties = null;
  if (injectOptions) {
    resolveInjections(injectOptions, ctx, checkDuplicateProperties);
  }
  if (methods) {
    for (const key in methods) {
      const methodHandler = methods[key];
      if (isFunction$1(methodHandler)) {
        {
          ctx[key] = methodHandler.bind(publicThis);
        }
      }
    }
  }
  if (dataOptions) {
    const data = dataOptions.call(publicThis, publicThis);
    if (!isObject$2(data)) ;
    else {
      instance.data = /* @__PURE__ */ reactive(data);
    }
  }
  shouldCacheAccess = true;
  if (computedOptions) {
    for (const key in computedOptions) {
      const opt = computedOptions[key];
      const get = isFunction$1(opt) ? opt.bind(publicThis, publicThis) : isFunction$1(opt.get) ? opt.get.bind(publicThis, publicThis) : NOOP;
      const set = !isFunction$1(opt) && isFunction$1(opt.set) ? opt.set.bind(publicThis) : NOOP;
      const c = computed({
        get,
        set
      });
      Object.defineProperty(ctx, key, {
        enumerable: true,
        configurable: true,
        get: () => c.value,
        set: (v) => c.value = v
      });
    }
  }
  if (watchOptions) {
    for (const key in watchOptions) {
      createWatcher(watchOptions[key], ctx, publicThis, key);
    }
  }
  if (provideOptions) {
    const provides = isFunction$1(provideOptions) ? provideOptions.call(publicThis) : provideOptions;
    Reflect.ownKeys(provides).forEach((key) => {
      provide(key, provides[key]);
    });
  }
  if (created) {
    callHook$1(created, instance, "c");
  }
  function registerLifecycleHook(register, hook) {
    if (isArray$1(hook)) {
      hook.forEach((_hook) => register(_hook.bind(publicThis)));
    } else if (hook) {
      register(hook.bind(publicThis));
    }
  }
  registerLifecycleHook(onBeforeMount, beforeMount);
  registerLifecycleHook(onMounted, mounted);
  registerLifecycleHook(onBeforeUpdate, beforeUpdate);
  registerLifecycleHook(onUpdated, updated);
  registerLifecycleHook(onActivated, activated);
  registerLifecycleHook(onDeactivated, deactivated);
  registerLifecycleHook(onErrorCaptured, errorCaptured);
  registerLifecycleHook(onRenderTracked, renderTracked);
  registerLifecycleHook(onRenderTriggered, renderTriggered);
  registerLifecycleHook(onBeforeUnmount, beforeUnmount);
  registerLifecycleHook(onUnmounted, unmounted);
  registerLifecycleHook(onServerPrefetch, serverPrefetch);
  if (isArray$1(expose)) {
    if (expose.length) {
      const exposed = instance.exposed || (instance.exposed = {});
      expose.forEach((key) => {
        Object.defineProperty(exposed, key, {
          get: () => publicThis[key],
          set: (val) => publicThis[key] = val,
          enumerable: true
        });
      });
    } else if (!instance.exposed) {
      instance.exposed = {};
    }
  }
  if (render2 && instance.render === NOOP) {
    instance.render = render2;
  }
  if (inheritAttrs != null) {
    instance.inheritAttrs = inheritAttrs;
  }
  if (components) instance.components = components;
  if (directives) instance.directives = directives;
  if (serverPrefetch) {
    markAsyncBoundary(instance);
  }
}
function resolveInjections(injectOptions, ctx, checkDuplicateProperties = NOOP) {
  if (isArray$1(injectOptions)) {
    injectOptions = normalizeInject(injectOptions);
  }
  for (const key in injectOptions) {
    const opt = injectOptions[key];
    let injected;
    if (isObject$2(opt)) {
      if ("default" in opt) {
        injected = inject(
          opt.from || key,
          opt.default,
          true
        );
      } else {
        injected = inject(opt.from || key);
      }
    } else {
      injected = inject(opt);
    }
    if (/* @__PURE__ */ isRef(injected)) {
      Object.defineProperty(ctx, key, {
        enumerable: true,
        configurable: true,
        get: () => injected.value,
        set: (v) => injected.value = v
      });
    } else {
      ctx[key] = injected;
    }
  }
}
function callHook$1(hook, instance, type) {
  callWithAsyncErrorHandling(
    isArray$1(hook) ? hook.map((h2) => h2.bind(instance.proxy)) : hook.bind(instance.proxy),
    instance,
    type
  );
}
function createWatcher(raw, ctx, publicThis, key) {
  let getter = key.includes(".") ? createPathGetter(publicThis, key) : () => publicThis[key];
  if (isString$2(raw)) {
    const handler = ctx[raw];
    if (isFunction$1(handler)) {
      {
        watch(getter, handler);
      }
    }
  } else if (isFunction$1(raw)) {
    {
      watch(getter, raw.bind(publicThis));
    }
  } else if (isObject$2(raw)) {
    if (isArray$1(raw)) {
      raw.forEach((r) => createWatcher(r, ctx, publicThis, key));
    } else {
      const handler = isFunction$1(raw.handler) ? raw.handler.bind(publicThis) : ctx[raw.handler];
      if (isFunction$1(handler)) {
        watch(getter, handler, raw);
      }
    }
  } else ;
}
function resolveMergedOptions(instance) {
  const base = instance.type;
  const { mixins, extends: extendsOptions } = base;
  const {
    mixins: globalMixins,
    optionsCache: cache2,
    config: { optionMergeStrategies }
  } = instance.appContext;
  const cached2 = cache2.get(base);
  let resolved;
  if (cached2) {
    resolved = cached2;
  } else if (!globalMixins.length && !mixins && !extendsOptions) {
    {
      resolved = base;
    }
  } else {
    resolved = {};
    if (globalMixins.length) {
      globalMixins.forEach(
        (m) => mergeOptions(resolved, m, optionMergeStrategies, true)
      );
    }
    mergeOptions(resolved, base, optionMergeStrategies);
  }
  if (isObject$2(base)) {
    cache2.set(base, resolved);
  }
  return resolved;
}
function mergeOptions(to, from, strats, asMixin = false) {
  const { mixins, extends: extendsOptions } = from;
  if (extendsOptions) {
    mergeOptions(to, extendsOptions, strats, true);
  }
  if (mixins) {
    mixins.forEach(
      (m) => mergeOptions(to, m, strats, true)
    );
  }
  for (const key in from) {
    if (asMixin && key === "expose") ;
    else {
      const strat = internalOptionMergeStrats[key] || strats && strats[key];
      to[key] = strat ? strat(to[key], from[key]) : from[key];
    }
  }
  return to;
}
const internalOptionMergeStrats = {
  data: mergeDataFn,
  props: mergeEmitsOrPropsOptions,
  emits: mergeEmitsOrPropsOptions,
  // objects
  methods: mergeObjectOptions,
  computed: mergeObjectOptions,
  // lifecycle
  beforeCreate: mergeAsArray,
  created: mergeAsArray,
  beforeMount: mergeAsArray,
  mounted: mergeAsArray,
  beforeUpdate: mergeAsArray,
  updated: mergeAsArray,
  beforeDestroy: mergeAsArray,
  beforeUnmount: mergeAsArray,
  destroyed: mergeAsArray,
  unmounted: mergeAsArray,
  activated: mergeAsArray,
  deactivated: mergeAsArray,
  errorCaptured: mergeAsArray,
  serverPrefetch: mergeAsArray,
  // assets
  components: mergeObjectOptions,
  directives: mergeObjectOptions,
  // watch
  watch: mergeWatchOptions,
  // provide / inject
  provide: mergeDataFn,
  inject: mergeInject
};
function mergeDataFn(to, from) {
  if (!from) {
    return to;
  }
  if (!to) {
    return from;
  }
  return function mergedDataFn() {
    return extend(
      isFunction$1(to) ? to.call(this, this) : to,
      isFunction$1(from) ? from.call(this, this) : from
    );
  };
}
function mergeInject(to, from) {
  return mergeObjectOptions(normalizeInject(to), normalizeInject(from));
}
function normalizeInject(raw) {
  if (isArray$1(raw)) {
    const res = {};
    for (let i = 0; i < raw.length; i++) {
      res[raw[i]] = raw[i];
    }
    return res;
  }
  return raw;
}
function mergeAsArray(to, from) {
  return to ? [...new Set([].concat(to, from))] : from;
}
function mergeObjectOptions(to, from) {
  return to ? extend(/* @__PURE__ */ Object.create(null), to, from) : from;
}
function mergeEmitsOrPropsOptions(to, from) {
  if (to) {
    if (isArray$1(to) && isArray$1(from)) {
      return [.../* @__PURE__ */ new Set([...to, ...from])];
    }
    return extend(
      /* @__PURE__ */ Object.create(null),
      normalizePropsOrEmits(to),
      normalizePropsOrEmits(from != null ? from : {})
    );
  } else {
    return from;
  }
}
function mergeWatchOptions(to, from) {
  if (!to) return from;
  if (!from) return to;
  const merged = extend(/* @__PURE__ */ Object.create(null), to);
  for (const key in from) {
    merged[key] = mergeAsArray(to[key], from[key]);
  }
  return merged;
}
function createAppContext() {
  return {
    app: null,
    config: {
      isNativeTag: NO,
      performance: false,
      globalProperties: {},
      optionMergeStrategies: {},
      errorHandler: void 0,
      warnHandler: void 0,
      compilerOptions: {}
    },
    mixins: [],
    components: {},
    directives: {},
    provides: /* @__PURE__ */ Object.create(null),
    optionsCache: /* @__PURE__ */ new WeakMap(),
    propsCache: /* @__PURE__ */ new WeakMap(),
    emitsCache: /* @__PURE__ */ new WeakMap()
  };
}
let uid$1 = 0;
function createAppAPI(render2, hydrate) {
  return function createApp2(rootComponent, rootProps = null) {
    if (!isFunction$1(rootComponent)) {
      rootComponent = extend({}, rootComponent);
    }
    if (rootProps != null && !isObject$2(rootProps)) {
      rootProps = null;
    }
    const context = createAppContext();
    const installedPlugins = /* @__PURE__ */ new WeakSet();
    const pluginCleanupFns = [];
    let isMounted = false;
    const app = context.app = {
      _uid: uid$1++,
      _component: rootComponent,
      _props: rootProps,
      _container: null,
      _context: context,
      _instance: null,
      version,
      get config() {
        return context.config;
      },
      set config(v) {
      },
      use(plugin, ...options) {
        if (installedPlugins.has(plugin)) ;
        else if (plugin && isFunction$1(plugin.install)) {
          installedPlugins.add(plugin);
          plugin.install(app, ...options);
        } else if (isFunction$1(plugin)) {
          installedPlugins.add(plugin);
          plugin(app, ...options);
        } else ;
        return app;
      },
      mixin(mixin) {
        {
          if (!context.mixins.includes(mixin)) {
            context.mixins.push(mixin);
          }
        }
        return app;
      },
      component(name, component) {
        if (!component) {
          return context.components[name];
        }
        context.components[name] = component;
        return app;
      },
      directive(name, directive) {
        if (!directive) {
          return context.directives[name];
        }
        context.directives[name] = directive;
        return app;
      },
      mount(rootContainer, isHydrate, namespace) {
        if (!isMounted) {
          const vnode = app._ceVNode || createVNode(rootComponent, rootProps);
          vnode.appContext = context;
          if (namespace === true) {
            namespace = "svg";
          } else if (namespace === false) {
            namespace = void 0;
          }
          {
            render2(vnode, rootContainer, namespace);
          }
          isMounted = true;
          app._container = rootContainer;
          rootContainer.__vue_app__ = app;
          return getComponentPublicInstance(vnode.component);
        }
      },
      onUnmount(cleanupFn) {
        pluginCleanupFns.push(cleanupFn);
      },
      unmount() {
        if (isMounted) {
          callWithAsyncErrorHandling(
            pluginCleanupFns,
            app._instance,
            16
          );
          render2(null, app._container);
          delete app._container.__vue_app__;
        }
      },
      provide(key, value) {
        context.provides[key] = value;
        return app;
      },
      runWithContext(fn) {
        const lastApp = currentApp;
        currentApp = app;
        try {
          return fn();
        } finally {
          currentApp = lastApp;
        }
      }
    };
    return app;
  };
}
let currentApp = null;
const getModelModifiers = (props, modelName) => {
  return modelName === "modelValue" || modelName === "model-value" ? props.modelModifiers : props[`${modelName}Modifiers`] || props[`${camelize(modelName)}Modifiers`] || props[`${hyphenate(modelName)}Modifiers`];
};
function emit(instance, event, ...rawArgs) {
  if (instance.isUnmounted) return;
  const props = instance.vnode.props || EMPTY_OBJ;
  let args = rawArgs;
  const isModelListener2 = event.startsWith("update:");
  const modifiers = isModelListener2 && getModelModifiers(props, event.slice(7));
  if (modifiers) {
    if (modifiers.trim) {
      args = rawArgs.map((a) => isString$2(a) ? a.trim() : a);
    }
    if (modifiers.number) {
      args = rawArgs.map(looseToNumber);
    }
  }
  let handlerName;
  let handler = props[handlerName = toHandlerKey(event)] || // also try camelCase event handler (#2249)
  props[handlerName = toHandlerKey(camelize(event))];
  if (!handler && isModelListener2) {
    handler = props[handlerName = toHandlerKey(hyphenate(event))];
  }
  if (handler) {
    callWithAsyncErrorHandling(
      handler,
      instance,
      6,
      args
    );
  }
  const onceHandler = props[handlerName + `Once`];
  if (onceHandler) {
    if (!instance.emitted) {
      instance.emitted = {};
    } else if (instance.emitted[handlerName]) {
      return;
    }
    instance.emitted[handlerName] = true;
    callWithAsyncErrorHandling(
      onceHandler,
      instance,
      6,
      args
    );
  }
}
const mixinEmitsCache = /* @__PURE__ */ new WeakMap();
function normalizeEmitsOptions(comp, appContext, asMixin = false) {
  const cache2 = asMixin ? mixinEmitsCache : appContext.emitsCache;
  const cached2 = cache2.get(comp);
  if (cached2 !== void 0) {
    return cached2;
  }
  const raw = comp.emits;
  let normalized = {};
  let hasExtends = false;
  if (!isFunction$1(comp)) {
    const extendEmits = (raw2) => {
      const normalizedFromExtend = normalizeEmitsOptions(raw2, appContext, true);
      if (normalizedFromExtend) {
        hasExtends = true;
        extend(normalized, normalizedFromExtend);
      }
    };
    if (!asMixin && appContext.mixins.length) {
      appContext.mixins.forEach(extendEmits);
    }
    if (comp.extends) {
      extendEmits(comp.extends);
    }
    if (comp.mixins) {
      comp.mixins.forEach(extendEmits);
    }
  }
  if (!raw && !hasExtends) {
    if (isObject$2(comp)) {
      cache2.set(comp, null);
    }
    return null;
  }
  if (isArray$1(raw)) {
    raw.forEach((key) => normalized[key] = null);
  } else {
    extend(normalized, raw);
  }
  if (isObject$2(comp)) {
    cache2.set(comp, normalized);
  }
  return normalized;
}
function isEmitListener(options, key) {
  if (!options || !isOn(key)) {
    return false;
  }
  key = key.slice(2).replace(/Once$/, "");
  return hasOwn$1(options, key[0].toLowerCase() + key.slice(1)) || hasOwn$1(options, hyphenate(key)) || hasOwn$1(options, key);
}
function markAttrsAccessed() {
}
function renderComponentRoot(instance) {
  const {
    type: Component,
    vnode,
    proxy,
    withProxy,
    propsOptions: [propsOptions],
    slots,
    attrs,
    emit: emit2,
    render: render2,
    renderCache,
    props,
    data,
    setupState,
    ctx,
    inheritAttrs
  } = instance;
  const prev = setCurrentRenderingInstance(instance);
  let result;
  let fallthroughAttrs;
  try {
    if (vnode.shapeFlag & 4) {
      const proxyToUse = withProxy || proxy;
      const thisProxy = false ? new Proxy(proxyToUse, {
        get(target, key, receiver) {
          warn$1(
            `Property '${String(
              key
            )}' was accessed via 'this'. Avoid using 'this' in templates.`
          );
          return Reflect.get(target, key, receiver);
        }
      }) : proxyToUse;
      result = normalizeVNode(
        render2.call(
          thisProxy,
          proxyToUse,
          renderCache,
          false ? /* @__PURE__ */ shallowReadonly(props) : props,
          setupState,
          data,
          ctx
        )
      );
      fallthroughAttrs = attrs;
    } else {
      const render22 = Component;
      if (false) ;
      result = normalizeVNode(
        render22.length > 1 ? render22(
          false ? /* @__PURE__ */ shallowReadonly(props) : props,
          false ? {
            get attrs() {
              markAttrsAccessed();
              return /* @__PURE__ */ shallowReadonly(attrs);
            },
            slots,
            emit: emit2
          } : { attrs, slots, emit: emit2 }
        ) : render22(
          false ? /* @__PURE__ */ shallowReadonly(props) : props,
          null
        )
      );
      fallthroughAttrs = Component.props ? attrs : getFunctionalFallthrough(attrs);
    }
  } catch (err) {
    blockStack.length = 0;
    handleError(err, instance, 1);
    result = createVNode(Comment);
  }
  let root = result;
  if (fallthroughAttrs && inheritAttrs !== false) {
    const keys = Object.keys(fallthroughAttrs);
    const { shapeFlag } = root;
    if (keys.length) {
      if (shapeFlag & (1 | 6)) {
        if (propsOptions && keys.some(isModelListener)) {
          fallthroughAttrs = filterModelListeners(
            fallthroughAttrs,
            propsOptions
          );
        }
        root = cloneVNode(root, fallthroughAttrs, false, true);
      }
    }
  }
  if (vnode.dirs) {
    root = cloneVNode(root, null, false, true);
    root.dirs = root.dirs ? root.dirs.concat(vnode.dirs) : vnode.dirs;
  }
  if (vnode.transition) {
    setTransitionHooks(root, vnode.transition);
  }
  {
    result = root;
  }
  setCurrentRenderingInstance(prev);
  return result;
}
const getFunctionalFallthrough = (attrs) => {
  let res;
  for (const key in attrs) {
    if (key === "class" || key === "style" || isOn(key)) {
      (res || (res = {}))[key] = attrs[key];
    }
  }
  return res;
};
const filterModelListeners = (attrs, props) => {
  const res = {};
  for (const key in attrs) {
    if (!isModelListener(key) || !(key.slice(9) in props)) {
      res[key] = attrs[key];
    }
  }
  return res;
};
function shouldUpdateComponent(prevVNode, nextVNode, optimized) {
  const { props: prevProps, children: prevChildren, component } = prevVNode;
  const { props: nextProps, children: nextChildren, patchFlag } = nextVNode;
  const emits = component.emitsOptions;
  if (nextVNode.dirs || nextVNode.transition) {
    return true;
  }
  if (optimized && patchFlag >= 0) {
    if (patchFlag & 1024) {
      return true;
    }
    if (patchFlag & 16) {
      if (!prevProps) {
        return !!nextProps;
      }
      return hasPropsChanged(prevProps, nextProps, emits);
    } else if (patchFlag & 8) {
      const dynamicProps = nextVNode.dynamicProps;
      for (let i = 0; i < dynamicProps.length; i++) {
        const key = dynamicProps[i];
        if (hasPropValueChanged(nextProps, prevProps, key) && !isEmitListener(emits, key)) {
          return true;
        }
      }
    }
  } else {
    if (prevChildren || nextChildren) {
      if (!nextChildren || !nextChildren.$stable) {
        return true;
      }
    }
    if (prevProps === nextProps) {
      return false;
    }
    if (!prevProps) {
      return !!nextProps;
    }
    if (!nextProps) {
      return true;
    }
    return hasPropsChanged(prevProps, nextProps, emits);
  }
  return false;
}
function hasPropsChanged(prevProps, nextProps, emitsOptions) {
  const nextKeys = Object.keys(nextProps);
  if (nextKeys.length !== Object.keys(prevProps).length) {
    return true;
  }
  for (let i = 0; i < nextKeys.length; i++) {
    const key = nextKeys[i];
    if (hasPropValueChanged(nextProps, prevProps, key) && !isEmitListener(emitsOptions, key)) {
      return true;
    }
  }
  return false;
}
function hasPropValueChanged(nextProps, prevProps, key) {
  const nextProp = nextProps[key];
  const prevProp = prevProps[key];
  if (key === "style" && isObject$2(nextProp) && isObject$2(prevProp)) {
    return !looseEqual(nextProp, prevProp);
  }
  return nextProp !== prevProp;
}
function updateHOCHostEl({ vnode, parent, suspense }, el) {
  while (parent) {
    const root = parent.subTree;
    if (root.suspense && root.suspense.activeBranch === vnode) {
      root.suspense.vnode.el = root.el = el;
      vnode = root;
    }
    if (root === vnode) {
      (vnode = parent.vnode).el = el;
      parent = parent.parent;
    } else {
      break;
    }
  }
  if (suspense && suspense.activeBranch === vnode) {
    suspense.vnode.el = el;
  }
}
const internalObjectProto = {};
const createInternalObject = () => Object.create(internalObjectProto);
const isInternalObject = (obj) => Object.getPrototypeOf(obj) === internalObjectProto;
function initProps(instance, rawProps, isStateful, isSSR = false) {
  const props = {};
  const attrs = createInternalObject();
  instance.propsDefaults = /* @__PURE__ */ Object.create(null);
  setFullProps(instance, rawProps, props, attrs);
  for (const key in instance.propsOptions[0]) {
    if (!(key in props)) {
      props[key] = void 0;
    }
  }
  if (isStateful) {
    instance.props = isSSR ? props : /* @__PURE__ */ shallowReactive(props);
  } else {
    if (!instance.type.props) {
      instance.props = attrs;
    } else {
      instance.props = props;
    }
  }
  instance.attrs = attrs;
}
function updateProps(instance, rawProps, rawPrevProps, optimized) {
  const {
    props,
    attrs,
    vnode: { patchFlag }
  } = instance;
  const rawCurrentProps = /* @__PURE__ */ toRaw(props);
  const [options] = instance.propsOptions;
  let hasAttrsChanged = false;
  if (
    // always force full diff in dev
    // - #1942 if hmr is enabled with sfc component
    // - vite#872 non-sfc component used by sfc component
    (optimized || patchFlag > 0) && !(patchFlag & 16)
  ) {
    if (patchFlag & 8) {
      const propsToUpdate = instance.vnode.dynamicProps;
      for (let i = 0; i < propsToUpdate.length; i++) {
        let key = propsToUpdate[i];
        if (isEmitListener(instance.emitsOptions, key)) {
          continue;
        }
        const value = rawProps[key];
        if (options) {
          if (hasOwn$1(attrs, key)) {
            if (value !== attrs[key]) {
              attrs[key] = value;
              hasAttrsChanged = true;
            }
          } else {
            const camelizedKey = camelize(key);
            props[camelizedKey] = resolvePropValue(
              options,
              rawCurrentProps,
              camelizedKey,
              value,
              instance,
              false
            );
          }
        } else {
          if (value !== attrs[key]) {
            attrs[key] = value;
            hasAttrsChanged = true;
          }
        }
      }
    }
  } else {
    if (setFullProps(instance, rawProps, props, attrs)) {
      hasAttrsChanged = true;
    }
    let kebabKey;
    for (const key in rawCurrentProps) {
      if (!rawProps || // for camelCase
      !hasOwn$1(rawProps, key) && // it's possible the original props was passed in as kebab-case
      // and converted to camelCase (#955)
      ((kebabKey = hyphenate(key)) === key || !hasOwn$1(rawProps, kebabKey))) {
        if (options) {
          if (rawPrevProps && // for camelCase
          (rawPrevProps[key] !== void 0 || // for kebab-case
          rawPrevProps[kebabKey] !== void 0)) {
            props[key] = resolvePropValue(
              options,
              rawCurrentProps,
              key,
              void 0,
              instance,
              true
            );
          }
        } else {
          delete props[key];
        }
      }
    }
    if (attrs !== rawCurrentProps) {
      for (const key in attrs) {
        if (!rawProps || !hasOwn$1(rawProps, key) && true) {
          delete attrs[key];
          hasAttrsChanged = true;
        }
      }
    }
  }
  if (hasAttrsChanged) {
    trigger(instance.attrs, "set", "");
  }
}
function setFullProps(instance, rawProps, props, attrs) {
  const [options, needCastKeys] = instance.propsOptions;
  let hasAttrsChanged = false;
  let rawCastValues;
  if (rawProps) {
    for (let key in rawProps) {
      if (isReservedProp(key)) {
        continue;
      }
      const value = rawProps[key];
      let camelKey;
      if (options && hasOwn$1(options, camelKey = camelize(key))) {
        if (!needCastKeys || !needCastKeys.includes(camelKey)) {
          props[camelKey] = value;
        } else {
          (rawCastValues || (rawCastValues = {}))[camelKey] = value;
        }
      } else if (!isEmitListener(instance.emitsOptions, key)) {
        if (!(key in attrs) || value !== attrs[key]) {
          attrs[key] = value;
          hasAttrsChanged = true;
        }
      }
    }
  }
  if (needCastKeys) {
    const rawCurrentProps = /* @__PURE__ */ toRaw(props);
    const castValues = rawCastValues || EMPTY_OBJ;
    for (let i = 0; i < needCastKeys.length; i++) {
      const key = needCastKeys[i];
      props[key] = resolvePropValue(
        options,
        rawCurrentProps,
        key,
        castValues[key],
        instance,
        !hasOwn$1(castValues, key)
      );
    }
  }
  return hasAttrsChanged;
}
function resolvePropValue(options, props, key, value, instance, isAbsent) {
  const opt = options[key];
  if (opt != null) {
    const hasDefault = hasOwn$1(opt, "default");
    if (hasDefault && value === void 0) {
      const defaultValue = opt.default;
      if (opt.type !== Function && !opt.skipFactory && isFunction$1(defaultValue)) {
        const { propsDefaults } = instance;
        if (key in propsDefaults) {
          value = propsDefaults[key];
        } else {
          const reset = setCurrentInstance(instance);
          value = propsDefaults[key] = defaultValue.call(
            null,
            props
          );
          reset();
        }
      } else {
        value = defaultValue;
      }
      if (instance.ce) {
        instance.ce._setProp(key, value);
      }
    }
    if (opt[
      0
      /* shouldCast */
    ]) {
      if (isAbsent && !hasDefault) {
        value = false;
      } else if (opt[
        1
        /* shouldCastTrue */
      ] && (value === "" || value === hyphenate(key))) {
        value = true;
      }
    }
  }
  return value;
}
const mixinPropsCache = /* @__PURE__ */ new WeakMap();
function normalizePropsOptions(comp, appContext, asMixin = false) {
  const cache2 = asMixin ? mixinPropsCache : appContext.propsCache;
  const cached2 = cache2.get(comp);
  if (cached2) {
    return cached2;
  }
  const raw = comp.props;
  const normalized = {};
  const needCastKeys = [];
  let hasExtends = false;
  if (!isFunction$1(comp)) {
    const extendProps = (raw2) => {
      hasExtends = true;
      const [props, keys] = normalizePropsOptions(raw2, appContext, true);
      extend(normalized, props);
      if (keys) needCastKeys.push(...keys);
    };
    if (!asMixin && appContext.mixins.length) {
      appContext.mixins.forEach(extendProps);
    }
    if (comp.extends) {
      extendProps(comp.extends);
    }
    if (comp.mixins) {
      comp.mixins.forEach(extendProps);
    }
  }
  if (!raw && !hasExtends) {
    if (isObject$2(comp)) {
      cache2.set(comp, EMPTY_ARR);
    }
    return EMPTY_ARR;
  }
  if (isArray$1(raw)) {
    for (let i = 0; i < raw.length; i++) {
      const normalizedKey = camelize(raw[i]);
      if (validatePropName(normalizedKey)) {
        normalized[normalizedKey] = EMPTY_OBJ;
      }
    }
  } else if (raw) {
    for (const key in raw) {
      const normalizedKey = camelize(key);
      if (validatePropName(normalizedKey)) {
        const opt = raw[key];
        const prop = normalized[normalizedKey] = isArray$1(opt) || isFunction$1(opt) ? { type: opt } : extend({}, opt);
        const propType = prop.type;
        let shouldCast = false;
        let shouldCastTrue = true;
        if (isArray$1(propType)) {
          for (let index = 0; index < propType.length; ++index) {
            const type = propType[index];
            const typeName = isFunction$1(type) && type.name;
            if (typeName === "Boolean") {
              shouldCast = true;
              break;
            } else if (typeName === "String") {
              shouldCastTrue = false;
            }
          }
        } else {
          shouldCast = isFunction$1(propType) && propType.name === "Boolean";
        }
        prop[
          0
          /* shouldCast */
        ] = shouldCast;
        prop[
          1
          /* shouldCastTrue */
        ] = shouldCastTrue;
        if (shouldCast || hasOwn$1(prop, "default")) {
          needCastKeys.push(normalizedKey);
        }
      }
    }
  }
  const res = [normalized, needCastKeys];
  if (isObject$2(comp)) {
    cache2.set(comp, res);
  }
  return res;
}
function validatePropName(key) {
  if (key[0] !== "$" && !isReservedProp(key)) {
    return true;
  }
  return false;
}
const isInternalKey = (key) => key === "_" || key === "_ctx" || key === "$stable";
const normalizeSlotValue = (value) => isArray$1(value) ? value.map(normalizeVNode) : [normalizeVNode(value)];
const normalizeSlot = (key, rawSlot, ctx) => {
  if (rawSlot._n) {
    return rawSlot;
  }
  const normalized = withCtx((...args) => {
    if (false) ;
    return normalizeSlotValue(rawSlot(...args));
  }, ctx);
  normalized._c = false;
  return normalized;
};
const normalizeObjectSlots = (rawSlots, slots, instance) => {
  const ctx = rawSlots._ctx;
  for (const key in rawSlots) {
    if (isInternalKey(key)) continue;
    const value = rawSlots[key];
    if (isFunction$1(value)) {
      slots[key] = normalizeSlot(key, value, ctx);
    } else if (value != null) {
      const normalized = normalizeSlotValue(value);
      slots[key] = () => normalized;
    }
  }
};
const normalizeVNodeSlots = (instance, children) => {
  const normalized = normalizeSlotValue(children);
  instance.slots.default = () => normalized;
};
const assignSlots = (slots, children, optimized) => {
  for (const key in children) {
    if (optimized || !isInternalKey(key)) {
      slots[key] = children[key];
    }
  }
};
const initSlots = (instance, children, optimized) => {
  const slots = instance.slots = createInternalObject();
  if (instance.vnode.shapeFlag & 32) {
    const type = children._;
    if (type) {
      assignSlots(slots, children, optimized);
      if (optimized) {
        def(slots, "_", type, true);
      }
    } else {
      normalizeObjectSlots(children, slots);
    }
  } else if (children) {
    normalizeVNodeSlots(instance, children);
  }
};
const updateSlots = (instance, children, optimized) => {
  const { vnode, slots } = instance;
  let needDeletionCheck = true;
  let deletionComparisonTarget = EMPTY_OBJ;
  if (vnode.shapeFlag & 32) {
    const type = children._;
    if (type) {
      if (optimized && type === 1) {
        needDeletionCheck = false;
      } else {
        assignSlots(slots, children, optimized);
      }
    } else {
      needDeletionCheck = !children.$stable;
      normalizeObjectSlots(children, slots);
    }
    deletionComparisonTarget = children;
  } else if (children) {
    normalizeVNodeSlots(instance, children);
    deletionComparisonTarget = { default: 1 };
  }
  if (needDeletionCheck) {
    for (const key in slots) {
      if (!isInternalKey(key) && deletionComparisonTarget[key] == null) {
        delete slots[key];
      }
    }
  }
};
const queuePostRenderEffect = queueEffectWithSuspense;
function createRenderer(options) {
  return baseCreateRenderer(options);
}
function baseCreateRenderer(options, createHydrationFns) {
  const target = getGlobalThis$1();
  target.__VUE__ = true;
  const {
    insert: hostInsert,
    remove: hostRemove,
    patchProp: hostPatchProp,
    createElement: hostCreateElement,
    createText: hostCreateText,
    createComment: hostCreateComment,
    setText: hostSetText,
    setElementText: hostSetElementText,
    parentNode: hostParentNode,
    nextSibling: hostNextSibling,
    setScopeId: hostSetScopeId = NOOP,
    insertStaticContent: hostInsertStaticContent
  } = options;
  const patch = (n1, n2, container, anchor = null, parentComponent = null, parentSuspense = null, namespace = void 0, slotScopeIds = null, optimized = !!n2.dynamicChildren) => {
    if (n1 === n2) {
      return;
    }
    if (n1 && !isSameVNodeType(n1, n2)) {
      anchor = getNextHostNode(n1);
      unmount(n1, parentComponent, parentSuspense, true);
      n1 = null;
    }
    if (n2.patchFlag === -2) {
      optimized = false;
      n2.dynamicChildren = null;
    }
    const { type, ref: ref3, shapeFlag } = n2;
    switch (type) {
      case Text:
        processText(n1, n2, container, anchor);
        break;
      case Comment:
        processCommentNode(n1, n2, container, anchor);
        break;
      case Static:
        if (n1 == null) {
          mountStaticNode(n2, container, anchor, namespace);
        }
        break;
      case Fragment:
        processFragment(
          n1,
          n2,
          container,
          anchor,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized
        );
        break;
      default:
        if (shapeFlag & 1) {
          processElement(
            n1,
            n2,
            container,
            anchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
        } else if (shapeFlag & 6) {
          processComponent(
            n1,
            n2,
            container,
            anchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
        } else if (shapeFlag & 64) {
          type.process(
            n1,
            n2,
            container,
            anchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized,
            internals
          );
        } else if (shapeFlag & 128) {
          type.process(
            n1,
            n2,
            container,
            anchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized,
            internals
          );
        } else ;
    }
    if (ref3 != null && parentComponent) {
      setRef(ref3, n1 && n1.ref, parentSuspense, n2 || n1, !n2);
    } else if (ref3 == null && n1 && n1.ref != null) {
      setRef(n1.ref, null, parentSuspense, n1, true);
    }
  };
  const processText = (n1, n2, container, anchor) => {
    if (n1 == null) {
      hostInsert(
        n2.el = hostCreateText(n2.children),
        container,
        anchor
      );
    } else {
      const el = n2.el = n1.el;
      if (n2.children !== n1.children) {
        hostSetText(el, n2.children);
      }
    }
  };
  const processCommentNode = (n1, n2, container, anchor) => {
    if (n1 == null) {
      hostInsert(
        n2.el = hostCreateComment(n2.children || ""),
        container,
        anchor
      );
    } else {
      n2.el = n1.el;
    }
  };
  const mountStaticNode = (n2, container, anchor, namespace) => {
    [n2.el, n2.anchor] = hostInsertStaticContent(
      n2.children,
      container,
      anchor,
      namespace,
      n2.el,
      n2.anchor
    );
  };
  const moveStaticNode = ({ el, anchor }, container, nextSibling) => {
    let next;
    while (el && el !== anchor) {
      next = hostNextSibling(el);
      hostInsert(el, container, nextSibling);
      el = next;
    }
    hostInsert(anchor, container, nextSibling);
  };
  const removeStaticNode = ({ el, anchor }) => {
    let next;
    while (el && el !== anchor) {
      next = hostNextSibling(el);
      hostRemove(el);
      el = next;
    }
    hostRemove(anchor);
  };
  const processElement = (n1, n2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
    if (n2.type === "svg") {
      namespace = "svg";
    } else if (n2.type === "math") {
      namespace = "mathml";
    }
    if (n1 == null) {
      mountElement(
        n2,
        container,
        anchor,
        parentComponent,
        parentSuspense,
        namespace,
        slotScopeIds,
        optimized
      );
    } else {
      const customElement = n1.el && n1.el._isVueCE ? n1.el : null;
      try {
        if (customElement) {
          customElement._beginPatch();
        }
        patchElement(
          n1,
          n2,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized
        );
      } finally {
        if (customElement) {
          customElement._endPatch();
        }
      }
    }
  };
  const mountElement = (vnode, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
    let el;
    let vnodeHook;
    const { props, shapeFlag, transition, dirs } = vnode;
    el = vnode.el = hostCreateElement(
      vnode.type,
      namespace,
      props && props.is,
      props
    );
    if (shapeFlag & 8) {
      hostSetElementText(el, vnode.children);
    } else if (shapeFlag & 16) {
      mountChildren(
        vnode.children,
        el,
        null,
        parentComponent,
        parentSuspense,
        resolveChildrenNamespace(vnode, namespace),
        slotScopeIds,
        optimized
      );
    }
    if (dirs) {
      invokeDirectiveHook(vnode, null, parentComponent, "created");
    }
    setScopeId(el, vnode, vnode.scopeId, slotScopeIds, parentComponent);
    if (props) {
      for (const key in props) {
        if (key !== "value" && !isReservedProp(key)) {
          hostPatchProp(el, key, null, props[key], namespace, parentComponent);
        }
      }
      if ("value" in props) {
        hostPatchProp(el, "value", null, props.value, namespace);
      }
      if (vnodeHook = props.onVnodeBeforeMount) {
        invokeVNodeHook(vnodeHook, parentComponent, vnode);
      }
    }
    if (dirs) {
      invokeDirectiveHook(vnode, null, parentComponent, "beforeMount");
    }
    const needCallTransitionHooks = needTransition(parentSuspense, transition);
    if (needCallTransitionHooks) {
      transition.beforeEnter(el);
    }
    hostInsert(el, container, anchor);
    if ((vnodeHook = props && props.onVnodeMounted) || needCallTransitionHooks || dirs) {
      queuePostRenderEffect(() => {
        try {
          vnodeHook && invokeVNodeHook(vnodeHook, parentComponent, vnode);
          needCallTransitionHooks && transition.enter(el);
          dirs && invokeDirectiveHook(vnode, null, parentComponent, "mounted");
        } finally {
        }
      }, parentSuspense);
    }
  };
  const setScopeId = (el, vnode, scopeId, slotScopeIds, parentComponent) => {
    if (scopeId) {
      hostSetScopeId(el, scopeId);
    }
    if (slotScopeIds) {
      for (let i = 0; i < slotScopeIds.length; i++) {
        hostSetScopeId(el, slotScopeIds[i]);
      }
    }
    if (parentComponent) {
      let subTree = parentComponent.subTree;
      if (vnode === subTree || isSuspense(subTree.type) && (subTree.ssContent === vnode || subTree.ssFallback === vnode)) {
        const parentVNode = parentComponent.vnode;
        setScopeId(
          el,
          parentVNode,
          parentVNode.scopeId,
          parentVNode.slotScopeIds,
          parentComponent.parent
        );
      }
    }
  };
  const mountChildren = (children, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized, start = 0) => {
    for (let i = start; i < children.length; i++) {
      const child = children[i] = optimized ? cloneIfMounted(children[i]) : normalizeVNode(children[i]);
      patch(
        null,
        child,
        container,
        anchor,
        parentComponent,
        parentSuspense,
        namespace,
        slotScopeIds,
        optimized
      );
    }
  };
  const patchElement = (n1, n2, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
    const el = n2.el = n1.el;
    let { patchFlag, dynamicChildren, dirs } = n2;
    patchFlag |= n1.patchFlag & 16;
    const oldProps = n1.props || EMPTY_OBJ;
    const newProps = n2.props || EMPTY_OBJ;
    let vnodeHook;
    parentComponent && toggleRecurse(parentComponent, false);
    if (vnodeHook = newProps.onVnodeBeforeUpdate) {
      invokeVNodeHook(vnodeHook, parentComponent, n2, n1);
    }
    if (dirs) {
      invokeDirectiveHook(n2, n1, parentComponent, "beforeUpdate");
    }
    parentComponent && toggleRecurse(parentComponent, true);
    if (oldProps.innerHTML && newProps.innerHTML == null || oldProps.textContent && newProps.textContent == null) {
      hostSetElementText(el, "");
    }
    if (dynamicChildren) {
      patchBlockChildren(
        n1.dynamicChildren,
        dynamicChildren,
        el,
        parentComponent,
        parentSuspense,
        resolveChildrenNamespace(n2, namespace),
        slotScopeIds
      );
    } else if (!optimized) {
      patchChildren(
        n1,
        n2,
        el,
        null,
        parentComponent,
        parentSuspense,
        resolveChildrenNamespace(n2, namespace),
        slotScopeIds,
        false
      );
    }
    if (patchFlag > 0) {
      if (patchFlag & 16) {
        patchProps(el, oldProps, newProps, parentComponent, namespace);
      } else {
        if (patchFlag & 2) {
          if (oldProps.class !== newProps.class) {
            hostPatchProp(el, "class", null, newProps.class, namespace);
          }
        }
        if (patchFlag & 4) {
          hostPatchProp(el, "style", oldProps.style, newProps.style, namespace);
        }
        if (patchFlag & 8) {
          const propsToUpdate = n2.dynamicProps;
          for (let i = 0; i < propsToUpdate.length; i++) {
            const key = propsToUpdate[i];
            const prev = oldProps[key];
            const next = newProps[key];
            if (next !== prev || key === "value") {
              hostPatchProp(el, key, prev, next, namespace, parentComponent);
            }
          }
        }
      }
      if (patchFlag & 1) {
        if (n1.children !== n2.children) {
          hostSetElementText(el, n2.children);
        }
      }
    } else if (!optimized && dynamicChildren == null) {
      patchProps(el, oldProps, newProps, parentComponent, namespace);
    }
    if ((vnodeHook = newProps.onVnodeUpdated) || dirs) {
      queuePostRenderEffect(() => {
        vnodeHook && invokeVNodeHook(vnodeHook, parentComponent, n2, n1);
        dirs && invokeDirectiveHook(n2, n1, parentComponent, "updated");
      }, parentSuspense);
    }
  };
  const patchBlockChildren = (oldChildren, newChildren, fallbackContainer, parentComponent, parentSuspense, namespace, slotScopeIds) => {
    for (let i = 0; i < newChildren.length; i++) {
      const oldVNode = oldChildren[i];
      const newVNode = newChildren[i];
      const container = (
        // oldVNode may be an errored async setup() component inside Suspense
        // which will not have a mounted element
        oldVNode.el && // - In the case of a Fragment, we need to provide the actual parent
        // of the Fragment itself so it can move its children.
        (oldVNode.type === Fragment || // - In the case of different nodes, there is going to be a replacement
        // which also requires the correct parent container
        !isSameVNodeType(oldVNode, newVNode) || // - In the case of a component, it could contain anything.
        oldVNode.shapeFlag & (6 | 64 | 128)) ? hostParentNode(oldVNode.el) : (
          // In other cases, the parent container is not actually used so we
          // just pass the block element here to avoid a DOM parentNode call.
          fallbackContainer
        )
      );
      patch(
        oldVNode,
        newVNode,
        container,
        null,
        parentComponent,
        parentSuspense,
        namespace,
        slotScopeIds,
        true
      );
    }
  };
  const patchProps = (el, oldProps, newProps, parentComponent, namespace) => {
    if (oldProps !== newProps) {
      if (oldProps !== EMPTY_OBJ) {
        for (const key in oldProps) {
          if (!isReservedProp(key) && !(key in newProps)) {
            hostPatchProp(
              el,
              key,
              oldProps[key],
              null,
              namespace,
              parentComponent
            );
          }
        }
      }
      for (const key in newProps) {
        if (isReservedProp(key)) continue;
        const next = newProps[key];
        const prev = oldProps[key];
        if (next !== prev && key !== "value") {
          hostPatchProp(el, key, prev, next, namespace, parentComponent);
        }
      }
      if ("value" in newProps) {
        hostPatchProp(el, "value", oldProps.value, newProps.value, namespace);
      }
    }
  };
  const processFragment = (n1, n2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
    const fragmentStartAnchor = n2.el = n1 ? n1.el : hostCreateText("");
    const fragmentEndAnchor = n2.anchor = n1 ? n1.anchor : hostCreateText("");
    let { patchFlag, dynamicChildren, slotScopeIds: fragmentSlotScopeIds } = n2;
    if (fragmentSlotScopeIds) {
      slotScopeIds = slotScopeIds ? slotScopeIds.concat(fragmentSlotScopeIds) : fragmentSlotScopeIds;
    }
    if (n1 == null) {
      hostInsert(fragmentStartAnchor, container, anchor);
      hostInsert(fragmentEndAnchor, container, anchor);
      mountChildren(
        // #10007
        // such fragment like `<></>` will be compiled into
        // a fragment which doesn't have a children.
        // In this case fallback to an empty array
        n2.children || [],
        container,
        fragmentEndAnchor,
        parentComponent,
        parentSuspense,
        namespace,
        slotScopeIds,
        optimized
      );
    } else {
      if (patchFlag > 0 && patchFlag & 64 && dynamicChildren && // #2715 the previous fragment could've been a BAILed one as a result
      // of renderSlot() with no valid children
      n1.dynamicChildren && n1.dynamicChildren.length === dynamicChildren.length) {
        patchBlockChildren(
          n1.dynamicChildren,
          dynamicChildren,
          container,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds
        );
        if (
          // #2080 if the stable fragment has a key, it's a <template v-for> that may
          //  get moved around. Make sure all root level vnodes inherit el.
          // #2134 or if it's a component root, it may also get moved around
          // as the component is being moved.
          n2.key != null || parentComponent && n2 === parentComponent.subTree
        ) {
          traverseStaticChildren(
            n1,
            n2,
            true
            /* shallow */
          );
        }
      } else {
        patchChildren(
          n1,
          n2,
          container,
          fragmentEndAnchor,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized
        );
      }
    }
  };
  const processComponent = (n1, n2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
    n2.slotScopeIds = slotScopeIds;
    if (n1 == null) {
      if (n2.shapeFlag & 512) {
        parentComponent.ctx.activate(
          n2,
          container,
          anchor,
          namespace,
          optimized
        );
      } else {
        mountComponent(
          n2,
          container,
          anchor,
          parentComponent,
          parentSuspense,
          namespace,
          optimized
        );
      }
    } else {
      updateComponent(n1, n2, optimized);
    }
  };
  const mountComponent = (initialVNode, container, anchor, parentComponent, parentSuspense, namespace, optimized) => {
    const instance = initialVNode.component = createComponentInstance(
      initialVNode,
      parentComponent,
      parentSuspense
    );
    if (isKeepAlive(initialVNode)) {
      instance.ctx.renderer = internals;
    }
    {
      setupComponent(instance, false, optimized);
    }
    if (instance.asyncDep) {
      parentSuspense && parentSuspense.registerDep(instance, setupRenderEffect, optimized);
      if (!initialVNode.el) {
        const placeholder = instance.subTree = createVNode(Comment);
        processCommentNode(null, placeholder, container, anchor);
        initialVNode.placeholder = placeholder.el;
      }
    } else {
      setupRenderEffect(
        instance,
        initialVNode,
        container,
        anchor,
        parentSuspense,
        namespace,
        optimized
      );
    }
  };
  const updateComponent = (n1, n2, optimized) => {
    const instance = n2.component = n1.component;
    if (shouldUpdateComponent(n1, n2, optimized)) {
      if (instance.asyncDep && !instance.asyncResolved) {
        updateComponentPreRender(instance, n2, optimized);
        return;
      } else {
        instance.next = n2;
        instance.update();
      }
    } else {
      n2.el = n1.el;
      instance.vnode = n2;
    }
  };
  const setupRenderEffect = (instance, initialVNode, container, anchor, parentSuspense, namespace, optimized) => {
    const componentUpdateFn = () => {
      if (!instance.isMounted) {
        let vnodeHook;
        const { el, props } = initialVNode;
        const { bm, m, parent, root, type } = instance;
        const isAsyncWrapperVNode = isAsyncWrapper(initialVNode);
        toggleRecurse(instance, false);
        if (bm) {
          invokeArrayFns(bm);
        }
        if (!isAsyncWrapperVNode && (vnodeHook = props && props.onVnodeBeforeMount)) {
          invokeVNodeHook(vnodeHook, parent, initialVNode);
        }
        toggleRecurse(instance, true);
        {
          if (root.ce && root.ce._hasShadowRoot()) {
            root.ce._injectChildStyle(
              type,
              instance.parent ? instance.parent.type : void 0
            );
          }
          const subTree = instance.subTree = renderComponentRoot(instance);
          patch(
            null,
            subTree,
            container,
            anchor,
            instance,
            parentSuspense,
            namespace
          );
          initialVNode.el = subTree.el;
        }
        if (m) {
          queuePostRenderEffect(m, parentSuspense);
        }
        if (!isAsyncWrapperVNode && (vnodeHook = props && props.onVnodeMounted)) {
          const scopedInitialVNode = initialVNode;
          queuePostRenderEffect(
            () => invokeVNodeHook(vnodeHook, parent, scopedInitialVNode),
            parentSuspense
          );
        }
        if (initialVNode.shapeFlag & 256 || parent && isAsyncWrapper(parent.vnode) && parent.vnode.shapeFlag & 256) {
          instance.a && queuePostRenderEffect(instance.a, parentSuspense);
        }
        instance.isMounted = true;
        initialVNode = container = anchor = null;
      } else {
        let { next, bu, u, parent, vnode } = instance;
        {
          const nonHydratedAsyncRoot = locateNonHydratedAsyncRoot(instance);
          if (nonHydratedAsyncRoot) {
            if (next) {
              next.el = vnode.el;
              updateComponentPreRender(instance, next, optimized);
            }
            nonHydratedAsyncRoot.asyncDep.then(() => {
              queuePostRenderEffect(() => {
                if (!instance.isUnmounted) update();
              }, parentSuspense);
            });
            return;
          }
        }
        let originNext = next;
        let vnodeHook;
        toggleRecurse(instance, false);
        if (next) {
          next.el = vnode.el;
          updateComponentPreRender(instance, next, optimized);
        } else {
          next = vnode;
        }
        if (bu) {
          invokeArrayFns(bu);
        }
        if (vnodeHook = next.props && next.props.onVnodeBeforeUpdate) {
          invokeVNodeHook(vnodeHook, parent, next, vnode);
        }
        toggleRecurse(instance, true);
        const nextTree = renderComponentRoot(instance);
        const prevTree = instance.subTree;
        instance.subTree = nextTree;
        patch(
          prevTree,
          nextTree,
          // parent may have changed if it's in a teleport
          hostParentNode(prevTree.el),
          // anchor may have changed if it's in a fragment
          getNextHostNode(prevTree),
          instance,
          parentSuspense,
          namespace
        );
        next.el = nextTree.el;
        if (originNext === null) {
          updateHOCHostEl(instance, nextTree.el);
        }
        if (u) {
          queuePostRenderEffect(u, parentSuspense);
        }
        if (vnodeHook = next.props && next.props.onVnodeUpdated) {
          queuePostRenderEffect(
            () => invokeVNodeHook(vnodeHook, parent, next, vnode),
            parentSuspense
          );
        }
      }
    };
    instance.scope.on();
    const effect2 = instance.effect = new ReactiveEffect(componentUpdateFn);
    instance.scope.off();
    const update = instance.update = effect2.run.bind(effect2);
    const job = instance.job = effect2.runIfDirty.bind(effect2);
    job.i = instance;
    job.id = instance.uid;
    effect2.scheduler = () => queueJob(job);
    toggleRecurse(instance, true);
    update();
  };
  const updateComponentPreRender = (instance, nextVNode, optimized) => {
    nextVNode.component = instance;
    const prevProps = instance.vnode.props;
    instance.vnode = nextVNode;
    instance.next = null;
    updateProps(instance, nextVNode.props, prevProps, optimized);
    updateSlots(instance, nextVNode.children, optimized);
    pauseTracking();
    flushPreFlushCbs(instance);
    resetTracking();
  };
  const patchChildren = (n1, n2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized = false) => {
    const c1 = n1 && n1.children;
    const prevShapeFlag = n1 ? n1.shapeFlag : 0;
    const c2 = n2.children;
    const { patchFlag, shapeFlag } = n2;
    if (patchFlag > 0) {
      if (patchFlag & 128) {
        patchKeyedChildren(
          c1,
          c2,
          container,
          anchor,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized
        );
        return;
      } else if (patchFlag & 256) {
        patchUnkeyedChildren(
          c1,
          c2,
          container,
          anchor,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized
        );
        return;
      }
    }
    if (shapeFlag & 8) {
      if (prevShapeFlag & 16) {
        unmountChildren(c1, parentComponent, parentSuspense);
      }
      if (c2 !== c1) {
        hostSetElementText(container, c2);
      }
    } else {
      if (prevShapeFlag & 16) {
        if (shapeFlag & 16) {
          patchKeyedChildren(
            c1,
            c2,
            container,
            anchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
        } else {
          unmountChildren(c1, parentComponent, parentSuspense, true);
        }
      } else {
        if (prevShapeFlag & 8) {
          hostSetElementText(container, "");
        }
        if (shapeFlag & 16) {
          mountChildren(
            c2,
            container,
            anchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
        }
      }
    }
  };
  const patchUnkeyedChildren = (c1, c2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
    c1 = c1 || EMPTY_ARR;
    c2 = c2 || EMPTY_ARR;
    const oldLength = c1.length;
    const newLength = c2.length;
    const commonLength = Math.min(oldLength, newLength);
    let i;
    for (i = 0; i < commonLength; i++) {
      const nextChild = c2[i] = optimized ? cloneIfMounted(c2[i]) : normalizeVNode(c2[i]);
      patch(
        c1[i],
        nextChild,
        container,
        null,
        parentComponent,
        parentSuspense,
        namespace,
        slotScopeIds,
        optimized
      );
    }
    if (oldLength > newLength) {
      unmountChildren(
        c1,
        parentComponent,
        parentSuspense,
        true,
        false,
        commonLength
      );
    } else {
      mountChildren(
        c2,
        container,
        anchor,
        parentComponent,
        parentSuspense,
        namespace,
        slotScopeIds,
        optimized,
        commonLength
      );
    }
  };
  const patchKeyedChildren = (c1, c2, container, parentAnchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
    let i = 0;
    const l2 = c2.length;
    let e1 = c1.length - 1;
    let e2 = l2 - 1;
    while (i <= e1 && i <= e2) {
      const n1 = c1[i];
      const n2 = c2[i] = optimized ? cloneIfMounted(c2[i]) : normalizeVNode(c2[i]);
      if (isSameVNodeType(n1, n2)) {
        patch(
          n1,
          n2,
          container,
          null,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized
        );
      } else {
        break;
      }
      i++;
    }
    while (i <= e1 && i <= e2) {
      const n1 = c1[e1];
      const n2 = c2[e2] = optimized ? cloneIfMounted(c2[e2]) : normalizeVNode(c2[e2]);
      if (isSameVNodeType(n1, n2)) {
        patch(
          n1,
          n2,
          container,
          null,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized
        );
      } else {
        break;
      }
      e1--;
      e2--;
    }
    if (i > e1) {
      if (i <= e2) {
        const nextPos = e2 + 1;
        const anchor = nextPos < l2 ? c2[nextPos].el : parentAnchor;
        while (i <= e2) {
          patch(
            null,
            c2[i] = optimized ? cloneIfMounted(c2[i]) : normalizeVNode(c2[i]),
            container,
            anchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
          i++;
        }
      }
    } else if (i > e2) {
      while (i <= e1) {
        unmount(c1[i], parentComponent, parentSuspense, true);
        i++;
      }
    } else {
      const s1 = i;
      const s2 = i;
      const keyToNewIndexMap = /* @__PURE__ */ new Map();
      for (i = s2; i <= e2; i++) {
        const nextChild = c2[i] = optimized ? cloneIfMounted(c2[i]) : normalizeVNode(c2[i]);
        if (nextChild.key != null) {
          keyToNewIndexMap.set(nextChild.key, i);
        }
      }
      let j;
      let patched = 0;
      const toBePatched = e2 - s2 + 1;
      let moved = false;
      let maxNewIndexSoFar = 0;
      const newIndexToOldIndexMap = new Array(toBePatched);
      for (i = 0; i < toBePatched; i++) newIndexToOldIndexMap[i] = 0;
      for (i = s1; i <= e1; i++) {
        const prevChild = c1[i];
        if (patched >= toBePatched) {
          unmount(prevChild, parentComponent, parentSuspense, true);
          continue;
        }
        let newIndex;
        if (prevChild.key != null) {
          newIndex = keyToNewIndexMap.get(prevChild.key);
        } else {
          for (j = s2; j <= e2; j++) {
            if (newIndexToOldIndexMap[j - s2] === 0 && isSameVNodeType(prevChild, c2[j])) {
              newIndex = j;
              break;
            }
          }
        }
        if (newIndex === void 0) {
          unmount(prevChild, parentComponent, parentSuspense, true);
        } else {
          newIndexToOldIndexMap[newIndex - s2] = i + 1;
          if (newIndex >= maxNewIndexSoFar) {
            maxNewIndexSoFar = newIndex;
          } else {
            moved = true;
          }
          patch(
            prevChild,
            c2[newIndex],
            container,
            null,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
          patched++;
        }
      }
      const increasingNewIndexSequence = moved ? getSequence(newIndexToOldIndexMap) : EMPTY_ARR;
      j = increasingNewIndexSequence.length - 1;
      for (i = toBePatched - 1; i >= 0; i--) {
        const nextIndex = s2 + i;
        const nextChild = c2[nextIndex];
        const anchorVNode = c2[nextIndex + 1];
        const anchor = nextIndex + 1 < l2 ? (
          // #13559, #14173 fallback to el placeholder for unresolved async component
          anchorVNode.el || resolveAsyncComponentPlaceholder(anchorVNode)
        ) : parentAnchor;
        if (newIndexToOldIndexMap[i] === 0) {
          patch(
            null,
            nextChild,
            container,
            anchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
        } else if (moved) {
          if (j < 0 || i !== increasingNewIndexSequence[j]) {
            move(nextChild, container, anchor, 2);
          } else {
            j--;
          }
        }
      }
    }
  };
  const move = (vnode, container, anchor, moveType, parentSuspense = null) => {
    const { el, type, transition, children, shapeFlag } = vnode;
    if (shapeFlag & 6) {
      move(vnode.component.subTree, container, anchor, moveType);
      return;
    }
    if (shapeFlag & 128) {
      vnode.suspense.move(container, anchor, moveType);
      return;
    }
    if (shapeFlag & 64) {
      type.move(vnode, container, anchor, internals);
      return;
    }
    if (type === Fragment) {
      hostInsert(el, container, anchor);
      for (let i = 0; i < children.length; i++) {
        move(children[i], container, anchor, moveType);
      }
      hostInsert(vnode.anchor, container, anchor);
      return;
    }
    if (type === Static) {
      moveStaticNode(vnode, container, anchor);
      return;
    }
    const needTransition2 = moveType !== 2 && shapeFlag & 1 && transition;
    if (needTransition2) {
      if (moveType === 0) {
        if (transition.persisted && !el[leaveCbKey]) {
          hostInsert(el, container, anchor);
        } else {
          transition.beforeEnter(el);
          hostInsert(el, container, anchor);
          queuePostRenderEffect(() => transition.enter(el), parentSuspense);
        }
      } else {
        const { leave, delayLeave, afterLeave } = transition;
        const remove22 = () => {
          if (vnode.ctx.isUnmounted) {
            hostRemove(el);
          } else {
            hostInsert(el, container, anchor);
          }
        };
        const performLeave = () => {
          const wasLeaving = el._isLeaving || !!el[leaveCbKey];
          if (el._isLeaving) {
            el[leaveCbKey](
              true
              /* cancelled */
            );
          }
          if (transition.persisted && !wasLeaving) {
            remove22();
          } else {
            leave(el, () => {
              remove22();
              afterLeave && afterLeave();
            });
          }
        };
        if (delayLeave) {
          delayLeave(el, remove22, performLeave);
        } else {
          performLeave();
        }
      }
    } else {
      hostInsert(el, container, anchor);
    }
  };
  const unmount = (vnode, parentComponent, parentSuspense, doRemove = false, optimized = false) => {
    const {
      type,
      props,
      ref: ref3,
      children,
      dynamicChildren,
      shapeFlag,
      patchFlag,
      dirs,
      cacheIndex,
      memo
    } = vnode;
    if (patchFlag === -2) {
      optimized = false;
    }
    if (ref3 != null) {
      pauseTracking();
      setRef(ref3, null, parentSuspense, vnode, true);
      resetTracking();
    }
    if (cacheIndex != null) {
      parentComponent.renderCache[cacheIndex] = void 0;
    }
    if (shapeFlag & 256) {
      parentComponent.ctx.deactivate(vnode);
      return;
    }
    const shouldInvokeDirs = shapeFlag & 1 && dirs;
    const shouldInvokeVnodeHook = !isAsyncWrapper(vnode);
    let vnodeHook;
    if (shouldInvokeVnodeHook && (vnodeHook = props && props.onVnodeBeforeUnmount)) {
      invokeVNodeHook(vnodeHook, parentComponent, vnode);
    }
    if (shapeFlag & 6) {
      unmountComponent(vnode.component, parentSuspense, doRemove);
    } else {
      if (shapeFlag & 128) {
        vnode.suspense.unmount(parentSuspense, doRemove);
        return;
      }
      if (shouldInvokeDirs) {
        invokeDirectiveHook(vnode, null, parentComponent, "beforeUnmount");
      }
      if (shapeFlag & 64) {
        vnode.type.remove(
          vnode,
          parentComponent,
          parentSuspense,
          internals,
          doRemove
        );
      } else if (dynamicChildren && // #5154
      // when v-once is used inside a block, setBlockTracking(-1) marks the
      // parent block with hasOnce: true
      // so that it doesn't take the fast path during unmount - otherwise
      // components nested in v-once are never unmounted.
      !dynamicChildren.hasOnce && // #1153: fast path should not be taken for non-stable (v-for) fragments
      (type !== Fragment || patchFlag > 0 && patchFlag & 64)) {
        unmountChildren(
          dynamicChildren,
          parentComponent,
          parentSuspense,
          false,
          true
        );
      } else if (type === Fragment && patchFlag & (128 | 256) || !optimized && shapeFlag & 16) {
        unmountChildren(children, parentComponent, parentSuspense);
      }
      if (doRemove) {
        remove2(vnode);
      }
    }
    const shouldInvalidateMemo = memo != null && cacheIndex == null;
    if (shouldInvokeVnodeHook && (vnodeHook = props && props.onVnodeUnmounted) || shouldInvokeDirs || shouldInvalidateMemo) {
      queuePostRenderEffect(() => {
        vnodeHook && invokeVNodeHook(vnodeHook, parentComponent, vnode);
        shouldInvokeDirs && invokeDirectiveHook(vnode, null, parentComponent, "unmounted");
        if (shouldInvalidateMemo) {
          vnode.el = null;
        }
      }, parentSuspense);
    }
  };
  const remove2 = (vnode) => {
    const { type, el, anchor, transition } = vnode;
    if (type === Fragment) {
      {
        removeFragment(el, anchor);
      }
      return;
    }
    if (type === Static) {
      removeStaticNode(vnode);
      return;
    }
    const performRemove = () => {
      hostRemove(el);
      if (transition && !transition.persisted && transition.afterLeave) {
        transition.afterLeave();
      }
    };
    if (vnode.shapeFlag & 1 && transition && !transition.persisted) {
      const { leave, delayLeave } = transition;
      const performLeave = () => leave(el, performRemove);
      if (delayLeave) {
        delayLeave(vnode.el, performRemove, performLeave);
      } else {
        performLeave();
      }
    } else {
      performRemove();
    }
  };
  const removeFragment = (cur, end) => {
    let next;
    while (cur !== end) {
      next = hostNextSibling(cur);
      hostRemove(cur);
      cur = next;
    }
    hostRemove(end);
  };
  const unmountComponent = (instance, parentSuspense, doRemove) => {
    const { bum, scope, job, subTree, um, m, a } = instance;
    invalidateMount(m);
    invalidateMount(a);
    if (bum) {
      invokeArrayFns(bum);
    }
    scope.stop();
    if (job) {
      job.flags |= 8;
      unmount(subTree, instance, parentSuspense, doRemove);
    }
    if (um) {
      queuePostRenderEffect(um, parentSuspense);
    }
    queuePostRenderEffect(() => {
      instance.isUnmounted = true;
    }, parentSuspense);
  };
  const unmountChildren = (children, parentComponent, parentSuspense, doRemove = false, optimized = false, start = 0) => {
    for (let i = start; i < children.length; i++) {
      unmount(children[i], parentComponent, parentSuspense, doRemove, optimized);
    }
  };
  const getNextHostNode = (vnode) => {
    if (vnode.shapeFlag & 6) {
      return getNextHostNode(vnode.component.subTree);
    }
    if (vnode.shapeFlag & 128) {
      return vnode.suspense.next();
    }
    const el = hostNextSibling(vnode.anchor || vnode.el);
    const teleportEnd = el && el[TeleportEndKey];
    return teleportEnd ? hostNextSibling(teleportEnd) : el;
  };
  let isFlushing = false;
  const render2 = (vnode, container, namespace) => {
    let instance;
    if (vnode == null) {
      if (container._vnode) {
        unmount(container._vnode, null, null, true);
        instance = container._vnode.component;
      }
    } else {
      patch(
        container._vnode || null,
        vnode,
        container,
        null,
        null,
        null,
        namespace
      );
    }
    container._vnode = vnode;
    if (!isFlushing) {
      isFlushing = true;
      flushPreFlushCbs(instance);
      flushPostFlushCbs();
      isFlushing = false;
    }
  };
  const internals = {
    p: patch,
    um: unmount,
    m: move,
    r: remove2,
    mt: mountComponent,
    mc: mountChildren,
    pc: patchChildren,
    pbc: patchBlockChildren,
    n: getNextHostNode,
    o: options
  };
  let hydrate;
  return {
    render: render2,
    hydrate,
    createApp: createAppAPI(render2)
  };
}
function resolveChildrenNamespace({ type, props }, currentNamespace) {
  return currentNamespace === "svg" && type === "foreignObject" || currentNamespace === "mathml" && type === "annotation-xml" && props && props.encoding && props.encoding.includes("html") ? void 0 : currentNamespace;
}
function toggleRecurse({ effect: effect2, job }, allowed) {
  if (allowed) {
    effect2.flags |= 32;
    job.flags |= 4;
  } else {
    effect2.flags &= -33;
    job.flags &= -5;
  }
}
function needTransition(parentSuspense, transition) {
  return (!parentSuspense || parentSuspense && !parentSuspense.pendingBranch) && transition && !transition.persisted;
}
function traverseStaticChildren(n1, n2, shallow = false) {
  const ch1 = n1.children;
  const ch2 = n2.children;
  if (isArray$1(ch1) && isArray$1(ch2)) {
    for (let i = 0; i < ch1.length; i++) {
      const c1 = ch1[i];
      let c2 = ch2[i];
      if (c2.shapeFlag & 1 && !c2.dynamicChildren) {
        if (c2.patchFlag <= 0 || c2.patchFlag === 32) {
          c2 = ch2[i] = cloneIfMounted(ch2[i]);
          c2.el = c1.el;
        }
        if (!shallow && c2.patchFlag !== -2)
          traverseStaticChildren(c1, c2);
      }
      if (c2.type === Text) {
        if (c2.patchFlag === -1) {
          c2 = ch2[i] = cloneIfMounted(c2);
        }
        c2.el = c1.el;
      }
      if (c2.type === Comment && !c2.el) {
        c2.el = c1.el;
      }
    }
  }
}
function getSequence(arr) {
  const p2 = arr.slice();
  const result = [0];
  let i, j, u, v, c;
  const len = arr.length;
  for (i = 0; i < len; i++) {
    const arrI = arr[i];
    if (arrI !== 0) {
      j = result[result.length - 1];
      if (arr[j] < arrI) {
        p2[i] = j;
        result.push(i);
        continue;
      }
      u = 0;
      v = result.length - 1;
      while (u < v) {
        c = u + v >> 1;
        if (arr[result[c]] < arrI) {
          u = c + 1;
        } else {
          v = c;
        }
      }
      if (arrI < arr[result[u]]) {
        if (u > 0) {
          p2[i] = result[u - 1];
        }
        result[u] = i;
      }
    }
  }
  u = result.length;
  v = result[u - 1];
  while (u-- > 0) {
    result[u] = v;
    v = p2[v];
  }
  return result;
}
function locateNonHydratedAsyncRoot(instance) {
  const subComponent = instance.subTree.component;
  if (subComponent) {
    if (subComponent.asyncDep && !subComponent.asyncResolved) {
      return subComponent;
    } else {
      return locateNonHydratedAsyncRoot(subComponent);
    }
  }
}
function invalidateMount(hooks) {
  if (hooks) {
    for (let i = 0; i < hooks.length; i++)
      hooks[i].flags |= 8;
  }
}
function resolveAsyncComponentPlaceholder(anchorVnode) {
  if (anchorVnode.placeholder) {
    return anchorVnode.placeholder;
  }
  const instance = anchorVnode.component;
  if (instance) {
    return resolveAsyncComponentPlaceholder(instance.subTree);
  }
  return null;
}
const isSuspense = (type) => type.__isSuspense;
function queueEffectWithSuspense(fn, suspense) {
  if (suspense && suspense.pendingBranch) {
    if (isArray$1(fn)) {
      suspense.effects.push(...fn);
    } else {
      suspense.effects.push(fn);
    }
  } else {
    queuePostFlushCb(fn);
  }
}
const Fragment = /* @__PURE__ */ Symbol.for("v-fgt");
const Text = /* @__PURE__ */ Symbol.for("v-txt");
const Comment = /* @__PURE__ */ Symbol.for("v-cmt");
const Static = /* @__PURE__ */ Symbol.for("v-stc");
const blockStack = [];
let currentBlock = null;
function openBlock(disableTracking = false) {
  blockStack.push(currentBlock = disableTracking ? null : []);
}
function closeBlock() {
  blockStack.pop();
  currentBlock = blockStack[blockStack.length - 1] || null;
}
let isBlockTreeEnabled = 1;
function setBlockTracking(value, inVOnce = false) {
  isBlockTreeEnabled += value;
  if (value < 0 && currentBlock && inVOnce) {
    currentBlock.hasOnce = true;
  }
}
function setupBlock(vnode) {
  vnode.dynamicChildren = isBlockTreeEnabled > 0 ? currentBlock || EMPTY_ARR : null;
  closeBlock();
  if (isBlockTreeEnabled > 0 && currentBlock) {
    currentBlock.push(vnode);
  }
  return vnode;
}
function createElementBlock(type, props, children, patchFlag, dynamicProps, shapeFlag) {
  return setupBlock(
    createBaseVNode(
      type,
      props,
      children,
      patchFlag,
      dynamicProps,
      shapeFlag,
      true
    )
  );
}
function createBlock(type, props, children, patchFlag, dynamicProps) {
  return setupBlock(
    createVNode(
      type,
      props,
      children,
      patchFlag,
      dynamicProps,
      true
    )
  );
}
function isVNode$1(value) {
  return value ? value.__v_isVNode === true : false;
}
function isSameVNodeType(n1, n2) {
  return n1.type === n2.type && n1.key === n2.key;
}
const normalizeKey = ({ key }) => key != null ? key : null;
const normalizeRef = ({
  ref: ref3,
  ref_key,
  ref_for
}) => {
  if (typeof ref3 === "number") {
    ref3 = "" + ref3;
  }
  return ref3 != null ? isString$2(ref3) || /* @__PURE__ */ isRef(ref3) || isFunction$1(ref3) ? { i: currentRenderingInstance, r: ref3, k: ref_key, f: !!ref_for } : ref3 : null;
};
function createBaseVNode(type, props = null, children = null, patchFlag = 0, dynamicProps = null, shapeFlag = type === Fragment ? 0 : 1, isBlockNode = false, needFullChildrenNormalization = false) {
  const vnode = {
    __v_isVNode: true,
    __v_skip: true,
    type,
    props,
    key: props && normalizeKey(props),
    ref: props && normalizeRef(props),
    scopeId: currentScopeId,
    slotScopeIds: null,
    children,
    component: null,
    suspense: null,
    ssContent: null,
    ssFallback: null,
    dirs: null,
    transition: null,
    el: null,
    anchor: null,
    target: null,
    targetStart: null,
    targetAnchor: null,
    staticCount: 0,
    shapeFlag,
    patchFlag,
    dynamicProps,
    dynamicChildren: null,
    appContext: null,
    ctx: currentRenderingInstance
  };
  if (needFullChildrenNormalization) {
    normalizeChildren(vnode, children);
    if (shapeFlag & 128) {
      type.normalize(vnode);
    }
  } else if (children) {
    vnode.shapeFlag |= isString$2(children) ? 8 : 16;
  }
  if (isBlockTreeEnabled > 0 && // avoid a block node from tracking itself
  !isBlockNode && // has current parent block
  currentBlock && // presence of a patch flag indicates this node needs patching on updates.
  // component nodes also should always be patched, because even if the
  // component doesn't need to update, it needs to persist the instance on to
  // the next vnode so that it can be properly unmounted later.
  (vnode.patchFlag > 0 || shapeFlag & 6) && // the EVENTS flag is only for hydration and if it is the only flag, the
  // vnode should not be considered dynamic due to handler caching.
  vnode.patchFlag !== 32) {
    currentBlock.push(vnode);
  }
  return vnode;
}
const createVNode = _createVNode;
function _createVNode(type, props = null, children = null, patchFlag = 0, dynamicProps = null, isBlockNode = false) {
  if (!type || type === NULL_DYNAMIC_COMPONENT) {
    type = Comment;
  }
  if (isVNode$1(type)) {
    const cloned = cloneVNode(
      type,
      props,
      true
      /* mergeRef: true */
    );
    if (children) {
      normalizeChildren(cloned, children);
    }
    if (isBlockTreeEnabled > 0 && !isBlockNode && currentBlock) {
      if (cloned.shapeFlag & 6) {
        currentBlock[currentBlock.indexOf(type)] = cloned;
      } else {
        currentBlock.push(cloned);
      }
    }
    cloned.patchFlag = -2;
    return cloned;
  }
  if (isClassComponent(type)) {
    type = type.__vccOpts;
  }
  if (props) {
    props = guardReactiveProps(props);
    let { class: klass, style } = props;
    if (klass && !isString$2(klass)) {
      props.class = normalizeClass(klass);
    }
    if (isObject$2(style)) {
      if (/* @__PURE__ */ isProxy(style) && !isArray$1(style)) {
        style = extend({}, style);
      }
      props.style = normalizeStyle(style);
    }
  }
  const shapeFlag = isString$2(type) ? 1 : isSuspense(type) ? 128 : isTeleport(type) ? 64 : isObject$2(type) ? 4 : isFunction$1(type) ? 2 : 0;
  return createBaseVNode(
    type,
    props,
    children,
    patchFlag,
    dynamicProps,
    shapeFlag,
    isBlockNode,
    true
  );
}
function guardReactiveProps(props) {
  if (!props) return null;
  return /* @__PURE__ */ isProxy(props) || isInternalObject(props) ? extend({}, props) : props;
}
function cloneVNode(vnode, extraProps, mergeRef = false, cloneTransition = false) {
  const { props, ref: ref3, patchFlag, children, transition } = vnode;
  const mergedProps = extraProps ? mergeProps(props || {}, extraProps) : props;
  const cloned = {
    __v_isVNode: true,
    __v_skip: true,
    type: vnode.type,
    props: mergedProps,
    key: mergedProps && normalizeKey(mergedProps),
    ref: extraProps && extraProps.ref ? (
      // #2078 in the case of <component :is="vnode" ref="extra"/>
      // if the vnode itself already has a ref, cloneVNode will need to merge
      // the refs so the single vnode can be set on multiple refs
      mergeRef && ref3 ? isArray$1(ref3) ? ref3.concat(normalizeRef(extraProps)) : [ref3, normalizeRef(extraProps)] : normalizeRef(extraProps)
    ) : ref3,
    scopeId: vnode.scopeId,
    slotScopeIds: vnode.slotScopeIds,
    children,
    target: vnode.target,
    targetStart: vnode.targetStart,
    targetAnchor: vnode.targetAnchor,
    staticCount: vnode.staticCount,
    shapeFlag: vnode.shapeFlag,
    // if the vnode is cloned with extra props, we can no longer assume its
    // existing patch flag to be reliable and need to add the FULL_PROPS flag.
    // note: preserve flag for fragments since they use the flag for children
    // fast paths only.
    patchFlag: extraProps && vnode.type !== Fragment ? patchFlag === -1 ? 16 : patchFlag | 16 : patchFlag,
    dynamicProps: vnode.dynamicProps,
    dynamicChildren: vnode.dynamicChildren,
    appContext: vnode.appContext,
    dirs: vnode.dirs,
    transition,
    // These should technically only be non-null on mounted VNodes. However,
    // they *should* be copied for kept-alive vnodes. So we just always copy
    // them since them being non-null during a mount doesn't affect the logic as
    // they will simply be overwritten.
    component: vnode.component,
    suspense: vnode.suspense,
    ssContent: vnode.ssContent && cloneVNode(vnode.ssContent),
    ssFallback: vnode.ssFallback && cloneVNode(vnode.ssFallback),
    placeholder: vnode.placeholder,
    el: vnode.el,
    anchor: vnode.anchor,
    ctx: vnode.ctx,
    ce: vnode.ce
  };
  if (transition && cloneTransition) {
    setTransitionHooks(
      cloned,
      transition.clone(cloned)
    );
  }
  return cloned;
}
function createTextVNode(text = " ", flag = 0) {
  return createVNode(Text, null, text, flag);
}
function createCommentVNode(text = "", asBlock = false) {
  return asBlock ? (openBlock(), createBlock(Comment, null, text)) : createVNode(Comment, null, text);
}
function normalizeVNode(child) {
  if (child == null || typeof child === "boolean") {
    return createVNode(Comment);
  } else if (isArray$1(child)) {
    return createVNode(
      Fragment,
      null,
      // #3666, avoid reference pollution when reusing vnode
      child.slice()
    );
  } else if (isVNode$1(child)) {
    return cloneIfMounted(child);
  } else {
    return createVNode(Text, null, String(child));
  }
}
function cloneIfMounted(child) {
  return child.el === null && child.patchFlag !== -1 || child.memo ? child : cloneVNode(child);
}
function normalizeChildren(vnode, children) {
  let type = 0;
  const { shapeFlag } = vnode;
  if (children == null) {
    children = null;
  } else if (isArray$1(children)) {
    type = 16;
  } else if (typeof children === "object") {
    if (shapeFlag & (1 | 64)) {
      const slot = children.default;
      if (slot) {
        slot._c && (slot._d = false);
        normalizeChildren(vnode, slot());
        slot._c && (slot._d = true);
      }
      return;
    } else {
      type = 32;
      const slotFlag = children._;
      if (!slotFlag && !isInternalObject(children)) {
        children._ctx = currentRenderingInstance;
      } else if (slotFlag === 3 && currentRenderingInstance) {
        if (currentRenderingInstance.slots._ === 1) {
          children._ = 1;
        } else {
          children._ = 2;
          vnode.patchFlag |= 1024;
        }
      }
    }
  } else if (isFunction$1(children)) {
    children = { default: children, _ctx: currentRenderingInstance };
    type = 32;
  } else {
    children = String(children);
    if (shapeFlag & 64) {
      type = 16;
      children = [createTextVNode(children)];
    } else {
      type = 8;
    }
  }
  vnode.children = children;
  vnode.shapeFlag |= type;
}
function mergeProps(...args) {
  const ret = {};
  for (let i = 0; i < args.length; i++) {
    const toMerge = args[i];
    for (const key in toMerge) {
      if (key === "class") {
        if (ret.class !== toMerge.class) {
          ret.class = normalizeClass([ret.class, toMerge.class]);
        }
      } else if (key === "style") {
        ret.style = normalizeStyle([ret.style, toMerge.style]);
      } else if (isOn(key)) {
        const existing = ret[key];
        const incoming = toMerge[key];
        if (incoming && existing !== incoming && !(isArray$1(existing) && existing.includes(incoming))) {
          ret[key] = existing ? [].concat(existing, incoming) : incoming;
        } else if (incoming == null && existing == null && // mergeProps({ 'onUpdate:modelValue': undefined }) should not retain
        // the model listener.
        !isModelListener(key)) {
          ret[key] = incoming;
        }
      } else if (key !== "") {
        ret[key] = toMerge[key];
      }
    }
  }
  return ret;
}
function invokeVNodeHook(hook, instance, vnode, prevVNode = null) {
  callWithAsyncErrorHandling(hook, instance, 7, [
    vnode,
    prevVNode
  ]);
}
const emptyAppContext = createAppContext();
let uid = 0;
function createComponentInstance(vnode, parent, suspense) {
  const type = vnode.type;
  const appContext = (parent ? parent.appContext : vnode.appContext) || emptyAppContext;
  const instance = {
    uid: uid++,
    vnode,
    type,
    parent,
    appContext,
    root: null,
    // to be immediately set
    next: null,
    subTree: null,
    // will be set synchronously right after creation
    effect: null,
    update: null,
    // will be set synchronously right after creation
    job: null,
    scope: new EffectScope(
      true
      /* detached */
    ),
    render: null,
    proxy: null,
    exposed: null,
    exposeProxy: null,
    withProxy: null,
    provides: parent ? parent.provides : Object.create(appContext.provides),
    ids: parent ? parent.ids : ["", 0, 0],
    accessCache: null,
    renderCache: [],
    // local resolved assets
    components: null,
    directives: null,
    // resolved props and emits options
    propsOptions: normalizePropsOptions(type, appContext),
    emitsOptions: normalizeEmitsOptions(type, appContext),
    // emit
    emit: null,
    // to be set immediately
    emitted: null,
    // props default value
    propsDefaults: EMPTY_OBJ,
    // inheritAttrs
    inheritAttrs: type.inheritAttrs,
    // state
    ctx: EMPTY_OBJ,
    data: EMPTY_OBJ,
    props: EMPTY_OBJ,
    attrs: EMPTY_OBJ,
    slots: EMPTY_OBJ,
    refs: EMPTY_OBJ,
    setupState: EMPTY_OBJ,
    setupContext: null,
    // suspense related
    suspense,
    suspenseId: suspense ? suspense.pendingId : 0,
    asyncDep: null,
    asyncResolved: false,
    // lifecycle hooks
    // not using enums here because it results in computed properties
    isMounted: false,
    isUnmounted: false,
    isDeactivated: false,
    bc: null,
    c: null,
    bm: null,
    m: null,
    bu: null,
    u: null,
    um: null,
    bum: null,
    da: null,
    a: null,
    rtg: null,
    rtc: null,
    ec: null,
    sp: null
  };
  {
    instance.ctx = { _: instance };
  }
  instance.root = parent ? parent.root : instance;
  instance.emit = emit.bind(null, instance);
  if (vnode.ce) {
    vnode.ce(instance);
  }
  return instance;
}
let currentInstance = null;
const getCurrentInstance = () => currentInstance || currentRenderingInstance;
let internalSetCurrentInstance;
let setInSSRSetupState;
{
  const g = getGlobalThis$1();
  const registerGlobalSetter = (key, setter) => {
    let setters;
    if (!(setters = g[key])) setters = g[key] = [];
    setters.push(setter);
    return (v) => {
      if (setters.length > 1) setters.forEach((set) => set(v));
      else setters[0](v);
    };
  };
  internalSetCurrentInstance = registerGlobalSetter(
    `__VUE_INSTANCE_SETTERS__`,
    (v) => currentInstance = v
  );
  setInSSRSetupState = registerGlobalSetter(
    `__VUE_SSR_SETTERS__`,
    (v) => isInSSRComponentSetup = v
  );
}
const setCurrentInstance = (instance) => {
  const prev = currentInstance;
  internalSetCurrentInstance(instance);
  instance.scope.on();
  return () => {
    instance.scope.off();
    internalSetCurrentInstance(prev);
  };
};
const unsetCurrentInstance = () => {
  currentInstance && currentInstance.scope.off();
  internalSetCurrentInstance(null);
};
function isStatefulComponent(instance) {
  return instance.vnode.shapeFlag & 4;
}
let isInSSRComponentSetup = false;
function setupComponent(instance, isSSR = false, optimized = false) {
  isSSR && setInSSRSetupState(isSSR);
  const { props, children } = instance.vnode;
  const isStateful = isStatefulComponent(instance);
  initProps(instance, props, isStateful, isSSR);
  initSlots(instance, children, optimized || isSSR);
  const setupResult = isStateful ? setupStatefulComponent(instance, isSSR) : void 0;
  isSSR && setInSSRSetupState(false);
  return setupResult;
}
function setupStatefulComponent(instance, isSSR) {
  const Component = instance.type;
  instance.accessCache = /* @__PURE__ */ Object.create(null);
  instance.proxy = new Proxy(instance.ctx, PublicInstanceProxyHandlers);
  const { setup } = Component;
  if (setup) {
    pauseTracking();
    const setupContext = instance.setupContext = setup.length > 1 ? createSetupContext(instance) : null;
    const reset = setCurrentInstance(instance);
    const setupResult = callWithErrorHandling(
      setup,
      instance,
      0,
      [
        instance.props,
        setupContext
      ]
    );
    const isAsyncSetup = isPromise$1(setupResult);
    resetTracking();
    reset();
    if ((isAsyncSetup || instance.sp) && !isAsyncWrapper(instance)) {
      markAsyncBoundary(instance);
    }
    if (isAsyncSetup) {
      setupResult.then(unsetCurrentInstance, unsetCurrentInstance);
      if (isSSR) {
        return setupResult.then((resolvedResult) => {
          handleSetupResult(instance, resolvedResult);
        }).catch((e) => {
          handleError(e, instance, 0);
        });
      } else {
        instance.asyncDep = setupResult;
      }
    } else {
      handleSetupResult(instance, setupResult);
    }
  } else {
    finishComponentSetup(instance);
  }
}
function handleSetupResult(instance, setupResult, isSSR) {
  if (isFunction$1(setupResult)) {
    if (instance.type.__ssrInlineRender) {
      instance.ssrRender = setupResult;
    } else {
      instance.render = setupResult;
    }
  } else if (isObject$2(setupResult)) {
    instance.setupState = proxyRefs(setupResult);
  } else ;
  finishComponentSetup(instance);
}
function finishComponentSetup(instance, isSSR, skipOptions) {
  const Component = instance.type;
  if (!instance.render) {
    instance.render = Component.render || NOOP;
  }
  {
    const reset = setCurrentInstance(instance);
    pauseTracking();
    try {
      applyOptions(instance);
    } finally {
      resetTracking();
      reset();
    }
  }
}
const attrsProxyHandlers = {
  get(target, key) {
    track(target, "get", "");
    return target[key];
  }
};
function createSetupContext(instance) {
  const expose = (exposed) => {
    instance.exposed = exposed || {};
  };
  {
    return {
      attrs: new Proxy(instance.attrs, attrsProxyHandlers),
      slots: instance.slots,
      emit: instance.emit,
      expose
    };
  }
}
function getComponentPublicInstance(instance) {
  if (instance.exposed) {
    return instance.exposeProxy || (instance.exposeProxy = new Proxy(proxyRefs(markRaw(instance.exposed)), {
      get(target, key) {
        if (key in target) {
          return target[key];
        } else if (key in publicPropertiesMap) {
          return publicPropertiesMap[key](instance);
        }
      },
      has(target, key) {
        return key in target || key in publicPropertiesMap;
      }
    }));
  } else {
    return instance.proxy;
  }
}
const classifyRE = /(?:^|[-_])\w/g;
const classify = (str) => str.replace(classifyRE, (c) => c.toUpperCase()).replace(/[-_]/g, "");
function getComponentName(Component, includeInferred = true) {
  return isFunction$1(Component) ? Component.displayName || Component.name : Component.name || includeInferred && Component.__name;
}
function formatComponentName(instance, Component, isRoot = false) {
  let name = getComponentName(Component);
  if (!name && Component.__file) {
    const match = Component.__file.match(/([^/\\]+)\.\w+$/);
    if (match) {
      name = match[1];
    }
  }
  if (!name && instance) {
    const inferFromRegistry = (registry) => {
      for (const key in registry) {
        if (registry[key] === Component) {
          return key;
        }
      }
    };
    name = inferFromRegistry(instance.components) || instance.parent && inferFromRegistry(
      instance.parent.type.components
    ) || inferFromRegistry(instance.appContext.components);
  }
  return name ? classify(name) : isRoot ? `App` : `Anonymous`;
}
function isClassComponent(value) {
  return isFunction$1(value) && "__vccOpts" in value;
}
const computed = (getterOrOptions, debugOptions) => {
  const c = /* @__PURE__ */ computed$1(getterOrOptions, debugOptions, isInSSRComponentSetup);
  return c;
};
function h(type, propsOrChildren, children) {
  try {
    setBlockTracking(-1);
    const l = arguments.length;
    if (l === 2) {
      if (isObject$2(propsOrChildren) && !isArray$1(propsOrChildren)) {
        if (isVNode$1(propsOrChildren)) {
          return createVNode(type, null, [propsOrChildren]);
        }
        return createVNode(type, propsOrChildren);
      } else {
        return createVNode(type, null, propsOrChildren);
      }
    } else {
      if (l > 3) {
        children = Array.prototype.slice.call(arguments, 2);
      } else if (l === 3 && isVNode$1(children)) {
        children = [children];
      }
      return createVNode(type, propsOrChildren, children);
    }
  } finally {
    setBlockTracking(1);
  }
}
const version = "3.5.38";
/**
* @vue/runtime-dom v3.5.38
* (c) 2018-present Yuxi (Evan) You and Vue contributors
* @license MIT
**/
let policy = void 0;
const tt = typeof window !== "undefined" && window.trustedTypes;
if (tt) {
  try {
    policy = /* @__PURE__ */ tt.createPolicy("vue", {
      createHTML: (val) => val
    });
  } catch (e) {
  }
}
const unsafeToTrustedHTML = policy ? (val) => policy.createHTML(val) : (val) => val;
const svgNS = "http://www.w3.org/2000/svg";
const mathmlNS = "http://www.w3.org/1998/Math/MathML";
const doc = typeof document !== "undefined" ? document : null;
const templateContainer = doc && /* @__PURE__ */ doc.createElement("template");
const nodeOps = {
  insert: (child, parent, anchor) => {
    parent.insertBefore(child, anchor || null);
  },
  remove: (child) => {
    const parent = child.parentNode;
    if (parent) {
      parent.removeChild(child);
    }
  },
  createElement: (tag, namespace, is, props) => {
    const el = namespace === "svg" ? doc.createElementNS(svgNS, tag) : namespace === "mathml" ? doc.createElementNS(mathmlNS, tag) : is ? doc.createElement(tag, { is }) : doc.createElement(tag);
    if (tag === "select" && props && props.multiple != null) {
      el.setAttribute("multiple", props.multiple);
    }
    return el;
  },
  createText: (text) => doc.createTextNode(text),
  createComment: (text) => doc.createComment(text),
  setText: (node, text) => {
    node.nodeValue = text;
  },
  setElementText: (el, text) => {
    el.textContent = text;
  },
  parentNode: (node) => node.parentNode,
  nextSibling: (node) => node.nextSibling,
  querySelector: (selector) => doc.querySelector(selector),
  setScopeId(el, id) {
    el.setAttribute(id, "");
  },
  // __UNSAFE__
  // Reason: innerHTML.
  // Static content here can only come from compiled templates.
  // As long as the user only uses trusted templates, this is safe.
  insertStaticContent(content, parent, anchor, namespace, start, end) {
    const before = anchor ? anchor.previousSibling : parent.lastChild;
    if (start && (start === end || start.nextSibling)) {
      while (true) {
        parent.insertBefore(start.cloneNode(true), anchor);
        if (start === end || !(start = start.nextSibling)) break;
      }
    } else {
      templateContainer.innerHTML = unsafeToTrustedHTML(
        namespace === "svg" ? `<svg>${content}</svg>` : namespace === "mathml" ? `<math>${content}</math>` : content
      );
      const template = templateContainer.content;
      if (namespace === "svg" || namespace === "mathml") {
        const wrapper = template.firstChild;
        while (wrapper.firstChild) {
          template.appendChild(wrapper.firstChild);
        }
        template.removeChild(wrapper);
      }
      parent.insertBefore(template, anchor);
    }
    return [
      // first
      before ? before.nextSibling : parent.firstChild,
      // last
      anchor ? anchor.previousSibling : parent.lastChild
    ];
  }
};
const TRANSITION = "transition";
const ANIMATION = "animation";
const vtcKey = /* @__PURE__ */ Symbol("_vtc");
const DOMTransitionPropsValidators = {
  name: String,
  type: String,
  css: {
    type: Boolean,
    default: true
  },
  duration: [String, Number, Object],
  enterFromClass: String,
  enterActiveClass: String,
  enterToClass: String,
  appearFromClass: String,
  appearActiveClass: String,
  appearToClass: String,
  leaveFromClass: String,
  leaveActiveClass: String,
  leaveToClass: String
};
const TransitionPropsValidators = /* @__PURE__ */ extend(
  {},
  BaseTransitionPropsValidators,
  DOMTransitionPropsValidators
);
const decorate$1 = (t) => {
  t.displayName = "Transition";
  t.props = TransitionPropsValidators;
  return t;
};
const Transition = /* @__PURE__ */ decorate$1(
  (props, { slots }) => h(BaseTransition, resolveTransitionProps(props), slots)
);
const callHook = (hook, args = []) => {
  if (isArray$1(hook)) {
    hook.forEach((h2) => h2(...args));
  } else if (hook) {
    hook(...args);
  }
};
const hasExplicitCallback = (hook) => {
  return hook ? isArray$1(hook) ? hook.some((h2) => h2.length > 1) : hook.length > 1 : false;
};
function resolveTransitionProps(rawProps) {
  const baseProps = {};
  for (const key in rawProps) {
    if (!(key in DOMTransitionPropsValidators)) {
      baseProps[key] = rawProps[key];
    }
  }
  if (rawProps.css === false) {
    return baseProps;
  }
  const {
    name = "v",
    type,
    duration,
    enterFromClass = `${name}-enter-from`,
    enterActiveClass = `${name}-enter-active`,
    enterToClass = `${name}-enter-to`,
    appearFromClass = enterFromClass,
    appearActiveClass = enterActiveClass,
    appearToClass = enterToClass,
    leaveFromClass = `${name}-leave-from`,
    leaveActiveClass = `${name}-leave-active`,
    leaveToClass = `${name}-leave-to`
  } = rawProps;
  const durations = normalizeDuration(duration);
  const enterDuration = durations && durations[0];
  const leaveDuration = durations && durations[1];
  const {
    onBeforeEnter,
    onEnter,
    onEnterCancelled,
    onLeave,
    onLeaveCancelled,
    onBeforeAppear = onBeforeEnter,
    onAppear = onEnter,
    onAppearCancelled = onEnterCancelled
  } = baseProps;
  const finishEnter = (el, isAppear, done, isCancelled) => {
    el._enterCancelled = isCancelled;
    removeTransitionClass(el, isAppear ? appearToClass : enterToClass);
    removeTransitionClass(el, isAppear ? appearActiveClass : enterActiveClass);
    done && done();
  };
  const finishLeave = (el, done) => {
    el._isLeaving = false;
    removeTransitionClass(el, leaveFromClass);
    removeTransitionClass(el, leaveToClass);
    removeTransitionClass(el, leaveActiveClass);
    done && done();
  };
  const makeEnterHook = (isAppear) => {
    return (el, done) => {
      const hook = isAppear ? onAppear : onEnter;
      const resolve2 = () => finishEnter(el, isAppear, done);
      callHook(hook, [el, resolve2]);
      nextFrame(() => {
        removeTransitionClass(el, isAppear ? appearFromClass : enterFromClass);
        addTransitionClass(el, isAppear ? appearToClass : enterToClass);
        if (!hasExplicitCallback(hook)) {
          whenTransitionEnds(el, type, enterDuration, resolve2);
        }
      });
    };
  };
  return extend(baseProps, {
    onBeforeEnter(el) {
      callHook(onBeforeEnter, [el]);
      addTransitionClass(el, enterFromClass);
      addTransitionClass(el, enterActiveClass);
    },
    onBeforeAppear(el) {
      callHook(onBeforeAppear, [el]);
      addTransitionClass(el, appearFromClass);
      addTransitionClass(el, appearActiveClass);
    },
    onEnter: makeEnterHook(false),
    onAppear: makeEnterHook(true),
    onLeave(el, done) {
      el._isLeaving = true;
      const resolve2 = () => finishLeave(el, done);
      addTransitionClass(el, leaveFromClass);
      if (!el._enterCancelled) {
        forceReflow(el);
        addTransitionClass(el, leaveActiveClass);
      } else {
        addTransitionClass(el, leaveActiveClass);
        forceReflow(el);
      }
      nextFrame(() => {
        if (!el._isLeaving) {
          return;
        }
        removeTransitionClass(el, leaveFromClass);
        addTransitionClass(el, leaveToClass);
        if (!hasExplicitCallback(onLeave)) {
          whenTransitionEnds(el, type, leaveDuration, resolve2);
        }
      });
      callHook(onLeave, [el, resolve2]);
    },
    onEnterCancelled(el) {
      finishEnter(el, false, void 0, true);
      callHook(onEnterCancelled, [el]);
    },
    onAppearCancelled(el) {
      finishEnter(el, true, void 0, true);
      callHook(onAppearCancelled, [el]);
    },
    onLeaveCancelled(el) {
      finishLeave(el);
      callHook(onLeaveCancelled, [el]);
    }
  });
}
function normalizeDuration(duration) {
  if (duration == null) {
    return null;
  } else if (isObject$2(duration)) {
    return [NumberOf(duration.enter), NumberOf(duration.leave)];
  } else {
    const n = NumberOf(duration);
    return [n, n];
  }
}
function NumberOf(val) {
  const res = toNumber(val);
  return res;
}
function addTransitionClass(el, cls) {
  cls.split(/\s+/).forEach((c) => c && el.classList.add(c));
  (el[vtcKey] || (el[vtcKey] = /* @__PURE__ */ new Set())).add(cls);
}
function removeTransitionClass(el, cls) {
  cls.split(/\s+/).forEach((c) => c && el.classList.remove(c));
  const _vtc = el[vtcKey];
  if (_vtc) {
    _vtc.delete(cls);
    if (!_vtc.size) {
      el[vtcKey] = void 0;
    }
  }
}
function nextFrame(cb) {
  requestAnimationFrame(() => {
    requestAnimationFrame(cb);
  });
}
let endId = 0;
function whenTransitionEnds(el, expectedType, explicitTimeout, resolve2) {
  const id = el._endId = ++endId;
  const resolveIfNotStale = () => {
    if (id === el._endId) {
      resolve2();
    }
  };
  if (explicitTimeout != null) {
    return setTimeout(resolveIfNotStale, explicitTimeout);
  }
  const { type, timeout, propCount } = getTransitionInfo(el, expectedType);
  if (!type) {
    return resolve2();
  }
  const endEvent = type + "end";
  let ended = 0;
  const end = () => {
    el.removeEventListener(endEvent, onEnd);
    resolveIfNotStale();
  };
  const onEnd = (e) => {
    if (e.target === el && ++ended >= propCount) {
      end();
    }
  };
  setTimeout(() => {
    if (ended < propCount) {
      end();
    }
  }, timeout + 1);
  el.addEventListener(endEvent, onEnd);
}
function getTransitionInfo(el, expectedType) {
  const styles = window.getComputedStyle(el);
  const getStyleProperties = (key) => (styles[key] || "").split(", ");
  const transitionDelays = getStyleProperties(`${TRANSITION}Delay`);
  const transitionDurations = getStyleProperties(`${TRANSITION}Duration`);
  const transitionTimeout = getTimeout(transitionDelays, transitionDurations);
  const animationDelays = getStyleProperties(`${ANIMATION}Delay`);
  const animationDurations = getStyleProperties(`${ANIMATION}Duration`);
  const animationTimeout = getTimeout(animationDelays, animationDurations);
  let type = null;
  let timeout = 0;
  let propCount = 0;
  if (expectedType === TRANSITION) {
    if (transitionTimeout > 0) {
      type = TRANSITION;
      timeout = transitionTimeout;
      propCount = transitionDurations.length;
    }
  } else if (expectedType === ANIMATION) {
    if (animationTimeout > 0) {
      type = ANIMATION;
      timeout = animationTimeout;
      propCount = animationDurations.length;
    }
  } else {
    timeout = Math.max(transitionTimeout, animationTimeout);
    type = timeout > 0 ? transitionTimeout > animationTimeout ? TRANSITION : ANIMATION : null;
    propCount = type ? type === TRANSITION ? transitionDurations.length : animationDurations.length : 0;
  }
  const hasTransform = type === TRANSITION && /\b(?:transform|all)(?:,|$)/.test(
    getStyleProperties(`${TRANSITION}Property`).toString()
  );
  return {
    type,
    timeout,
    propCount,
    hasTransform
  };
}
function getTimeout(delays, durations) {
  while (delays.length < durations.length) {
    delays = delays.concat(delays);
  }
  return Math.max(...durations.map((d, i) => toMs(d) + toMs(delays[i])));
}
function toMs(s) {
  if (s === "auto") return 0;
  return Number(s.slice(0, -1).replace(",", ".")) * 1e3;
}
function forceReflow(el) {
  const targetDocument = el ? el.ownerDocument : document;
  return targetDocument.body.offsetHeight;
}
function patchClass(el, value, isSVG) {
  const transitionClasses = el[vtcKey];
  if (transitionClasses) {
    value = (value ? [value, ...transitionClasses] : [...transitionClasses]).join(" ");
  }
  if (value == null) {
    el.removeAttribute("class");
  } else if (isSVG) {
    el.setAttribute("class", value);
  } else {
    el.className = value;
  }
}
const vShowOriginalDisplay = /* @__PURE__ */ Symbol("_vod");
const vShowHidden = /* @__PURE__ */ Symbol("_vsh");
const CSS_VAR_TEXT = /* @__PURE__ */ Symbol("");
const displayRE = /(?:^|;)\s*display\s*:/;
function patchStyle(el, prev, next) {
  const style = el.style;
  const isCssString = isString$2(next);
  let hasControlledDisplay = false;
  if (next && !isCssString) {
    if (prev) {
      if (!isString$2(prev)) {
        for (const key in prev) {
          if (next[key] == null) {
            setStyle(style, key, "");
          }
        }
      } else {
        for (const prevStyle of prev.split(";")) {
          const key = prevStyle.slice(0, prevStyle.indexOf(":")).trim();
          if (next[key] == null) {
            setStyle(style, key, "");
          }
        }
      }
    }
    for (const key in next) {
      if (key === "display") {
        hasControlledDisplay = true;
      }
      const value = next[key];
      if (value != null) {
        if (!shouldPreserveTextareaResizeStyle(
          el,
          key,
          !isString$2(prev) && prev ? prev[key] : void 0,
          value
        )) {
          setStyle(style, key, value);
        }
      } else {
        setStyle(style, key, "");
      }
    }
  } else {
    if (isCssString) {
      if (prev !== next) {
        const cssVarText = style[CSS_VAR_TEXT];
        if (cssVarText) {
          next += ";" + cssVarText;
        }
        style.cssText = next;
        hasControlledDisplay = displayRE.test(next);
      }
    } else if (prev) {
      el.removeAttribute("style");
    }
  }
  if (vShowOriginalDisplay in el) {
    el[vShowOriginalDisplay] = hasControlledDisplay ? style.display : "";
    if (el[vShowHidden]) {
      style.display = "none";
    }
  }
}
const importantRE = /\s*!important$/;
function setStyle(style, name, val) {
  if (isArray$1(val)) {
    val.forEach((v) => setStyle(style, name, v));
  } else {
    if (val == null) val = "";
    if (name.startsWith("--")) {
      style.setProperty(name, val);
    } else {
      const prefixed = autoPrefix(style, name);
      if (importantRE.test(val)) {
        style.setProperty(
          hyphenate(prefixed),
          val.replace(importantRE, ""),
          "important"
        );
      } else {
        style[prefixed] = val;
      }
    }
  }
}
const prefixes = ["Webkit", "Moz", "ms"];
const prefixCache = {};
function autoPrefix(style, rawName) {
  const cached2 = prefixCache[rawName];
  if (cached2) {
    return cached2;
  }
  let name = camelize(rawName);
  if (name !== "filter" && name in style) {
    return prefixCache[rawName] = name;
  }
  name = capitalize$1(name);
  for (let i = 0; i < prefixes.length; i++) {
    const prefixed = prefixes[i] + name;
    if (prefixed in style) {
      return prefixCache[rawName] = prefixed;
    }
  }
  return rawName;
}
function shouldPreserveTextareaResizeStyle(el, key, prev, next) {
  return el.tagName === "TEXTAREA" && (key === "width" || key === "height") && isString$2(next) && prev === next;
}
const xlinkNS = "http://www.w3.org/1999/xlink";
function patchAttr(el, key, value, isSVG, instance, isBoolean2 = isSpecialBooleanAttr(key)) {
  if (isSVG && key.startsWith("xlink:")) {
    if (value == null) {
      el.removeAttributeNS(xlinkNS, key.slice(6, key.length));
    } else {
      el.setAttributeNS(xlinkNS, key, value);
    }
  } else {
    if (value == null || isBoolean2 && !includeBooleanAttr(value)) {
      el.removeAttribute(key);
    } else {
      el.setAttribute(
        key,
        isBoolean2 ? "" : isSymbol(value) ? String(value) : value
      );
    }
  }
}
function patchDOMProp(el, key, value, parentComponent, attrName) {
  if (key === "innerHTML" || key === "textContent") {
    if (value != null) {
      el[key] = key === "innerHTML" ? unsafeToTrustedHTML(value) : value;
    }
    return;
  }
  const tag = el.tagName;
  if (key === "value" && tag !== "PROGRESS" && // custom elements may use _value internally
  !tag.includes("-")) {
    const oldValue = tag === "OPTION" ? el.getAttribute("value") || "" : el.value;
    const newValue = value == null ? (
      // #11647: value should be set as empty string for null and undefined,
      // but <input type="checkbox"> should be set as 'on'.
      el.type === "checkbox" ? "on" : ""
    ) : String(value);
    if (oldValue !== newValue || !("_value" in el)) {
      el.value = newValue;
    }
    if (value == null) {
      el.removeAttribute(key);
    }
    el._value = value;
    return;
  }
  let needRemove = false;
  if (value === "" || value == null) {
    const type = typeof el[key];
    if (type === "boolean") {
      value = includeBooleanAttr(value);
    } else if (value == null && type === "string") {
      value = "";
      needRemove = true;
    } else if (type === "number") {
      value = 0;
      needRemove = true;
    }
  }
  try {
    el[key] = value;
  } catch (e) {
  }
  needRemove && el.removeAttribute(attrName || key);
}
function addEventListener(el, event, handler, options) {
  el.addEventListener(event, handler, options);
}
function removeEventListener(el, event, handler, options) {
  el.removeEventListener(event, handler, options);
}
const veiKey = /* @__PURE__ */ Symbol("_vei");
function patchEvent(el, rawName, prevValue, nextValue, instance = null) {
  const invokers = el[veiKey] || (el[veiKey] = {});
  const existingInvoker = invokers[rawName];
  if (nextValue && existingInvoker) {
    existingInvoker.value = nextValue;
  } else {
    const [name, options] = parseName(rawName);
    if (nextValue) {
      const invoker = invokers[rawName] = createInvoker(
        nextValue,
        instance
      );
      addEventListener(el, name, invoker, options);
    } else if (existingInvoker) {
      removeEventListener(el, name, existingInvoker, options);
      invokers[rawName] = void 0;
    }
  }
}
const optionsModifierRE = /(?:Once|Passive|Capture)$/;
function parseName(name) {
  let options;
  if (optionsModifierRE.test(name)) {
    options = {};
    let m;
    while (m = name.match(optionsModifierRE)) {
      name = name.slice(0, name.length - m[0].length);
      options[m[0].toLowerCase()] = true;
    }
  }
  const event = name[2] === ":" ? name.slice(3) : hyphenate(name.slice(2));
  return [event, options];
}
let cachedNow = 0;
const p = /* @__PURE__ */ Promise.resolve();
const getNow = () => cachedNow || (p.then(() => cachedNow = 0), cachedNow = Date.now());
function createInvoker(initialValue, instance) {
  const invoker = (e) => {
    if (!e._vts) {
      e._vts = Date.now();
    } else if (e._vts <= invoker.attached) {
      return;
    }
    const value = invoker.value;
    if (isArray$1(value)) {
      const originalStop = e.stopImmediatePropagation;
      e.stopImmediatePropagation = () => {
        originalStop.call(e);
        e._stopped = true;
      };
      const handlers = value.slice();
      const args = [e];
      for (let i = 0; i < handlers.length; i++) {
        if (e._stopped) {
          break;
        }
        const handler = handlers[i];
        if (handler) {
          callWithAsyncErrorHandling(
            handler,
            instance,
            5,
            args
          );
        }
      }
    } else {
      callWithAsyncErrorHandling(
        value,
        instance,
        5,
        [e]
      );
    }
  };
  invoker.value = initialValue;
  invoker.attached = getNow();
  return invoker;
}
const isNativeOn = (key) => key.charCodeAt(0) === 111 && key.charCodeAt(1) === 110 && // lowercase letter
key.charCodeAt(2) > 96 && key.charCodeAt(2) < 123;
const patchProp = (el, key, prevValue, nextValue, namespace, parentComponent) => {
  const isSVG = namespace === "svg";
  if (key === "class") {
    patchClass(el, nextValue, isSVG);
  } else if (key === "style") {
    patchStyle(el, prevValue, nextValue);
  } else if (isOn(key)) {
    if (!isModelListener(key)) {
      patchEvent(el, key, prevValue, nextValue, parentComponent);
    }
  } else if (key[0] === "." ? (key = key.slice(1), true) : key[0] === "^" ? (key = key.slice(1), false) : shouldSetAsProp(el, key, nextValue, isSVG)) {
    patchDOMProp(el, key, nextValue);
    if (!el.tagName.includes("-") && (key === "value" || key === "checked" || key === "selected")) {
      patchAttr(el, key, nextValue, isSVG, parentComponent, key !== "value");
    }
  } else if (
    // #11081 force set props for possible async custom element
    el._isVueCE && // #12408 check if it's declared prop or it's async custom element
    (shouldSetAsPropForVueCE(el, key) || // @ts-expect-error _def is private
    el._def.__asyncLoader && (/[A-Z]/.test(key) || !isString$2(nextValue)))
  ) {
    patchDOMProp(el, camelize(key), nextValue, parentComponent, key);
  } else {
    if (key === "true-value") {
      el._trueValue = nextValue;
    } else if (key === "false-value") {
      el._falseValue = nextValue;
    }
    patchAttr(el, key, nextValue, isSVG);
  }
};
function shouldSetAsProp(el, key, value, isSVG) {
  if (isSVG) {
    if (key === "innerHTML" || key === "textContent") {
      return true;
    }
    if (key in el && isNativeOn(key) && isFunction$1(value)) {
      return true;
    }
    return false;
  }
  if (key === "spellcheck" || key === "draggable" || key === "translate" || key === "autocorrect") {
    return false;
  }
  if (key === "sandbox" && el.tagName === "IFRAME") {
    return false;
  }
  if (key === "form") {
    return false;
  }
  if (key === "list" && el.tagName === "INPUT") {
    return false;
  }
  if (key === "type" && el.tagName === "TEXTAREA") {
    return false;
  }
  if (key === "width" || key === "height") {
    const tag = el.tagName;
    if (tag === "IMG" || tag === "VIDEO" || tag === "CANVAS" || tag === "SOURCE") {
      return false;
    }
  }
  if (isNativeOn(key) && isString$2(value)) {
    return false;
  }
  return key in el;
}
function shouldSetAsPropForVueCE(el, key) {
  const props = (
    // @ts-expect-error _def is private
    el._def.props
  );
  if (!props) {
    return false;
  }
  const camelKey = camelize(key);
  return Array.isArray(props) ? props.some((prop) => camelize(prop) === camelKey) : Object.keys(props).some((prop) => camelize(prop) === camelKey);
}
const positionMap = /* @__PURE__ */ new WeakMap();
const newPositionMap = /* @__PURE__ */ new WeakMap();
const moveCbKey = /* @__PURE__ */ Symbol("_moveCb");
const enterCbKey = /* @__PURE__ */ Symbol("_enterCb");
const decorate = (t) => {
  delete t.props.mode;
  return t;
};
const TransitionGroupImpl = /* @__PURE__ */ decorate({
  name: "TransitionGroup",
  props: /* @__PURE__ */ extend({}, TransitionPropsValidators, {
    tag: String,
    moveClass: String
  }),
  setup(props, { slots }) {
    const instance = getCurrentInstance();
    const state = useTransitionState();
    let prevChildren;
    let children;
    onUpdated(() => {
      if (!prevChildren.length) {
        return;
      }
      const moveClass = props.moveClass || `${props.name || "v"}-move`;
      if (!hasCSSTransform(
        prevChildren[0].el,
        instance.vnode.el,
        moveClass
      )) {
        prevChildren = [];
        return;
      }
      prevChildren.forEach(callPendingCbs);
      prevChildren.forEach(recordPosition);
      const movedChildren = prevChildren.filter(applyTranslation);
      forceReflow(instance.vnode.el);
      movedChildren.forEach((c) => {
        const el = c.el;
        const style = el.style;
        addTransitionClass(el, moveClass);
        style.transform = style.webkitTransform = style.transitionDuration = "";
        const cb = el[moveCbKey] = (e) => {
          if (e && e.target !== el) {
            return;
          }
          if (!e || e.propertyName.endsWith("transform")) {
            el.removeEventListener("transitionend", cb);
            el[moveCbKey] = null;
            removeTransitionClass(el, moveClass);
          }
        };
        el.addEventListener("transitionend", cb);
      });
      prevChildren = [];
    });
    return () => {
      const rawProps = /* @__PURE__ */ toRaw(props);
      const cssTransitionProps = resolveTransitionProps(rawProps);
      let tag = rawProps.tag || Fragment;
      prevChildren = [];
      if (children) {
        for (let i = 0; i < children.length; i++) {
          const child = children[i];
          if (child.el && child.el instanceof Element && // Hidden v-show nodes have no previous layout box to animate from.
          !child.el[vShowHidden]) {
            prevChildren.push(child);
            setTransitionHooks(
              child,
              resolveTransitionHooks(
                child,
                cssTransitionProps,
                state,
                instance
              )
            );
            positionMap.set(child, getPosition(child.el));
          }
        }
      }
      children = slots.default ? getTransitionRawChildren(slots.default()) : [];
      for (let i = 0; i < children.length; i++) {
        const child = children[i];
        if (child.key != null) {
          setTransitionHooks(
            child,
            resolveTransitionHooks(child, cssTransitionProps, state, instance)
          );
        }
      }
      return createVNode(tag, null, children);
    };
  }
});
const TransitionGroup = TransitionGroupImpl;
function callPendingCbs(c) {
  const el = c.el;
  if (el[moveCbKey]) {
    el[moveCbKey]();
  }
  if (el[enterCbKey]) {
    el[enterCbKey]();
  }
}
function recordPosition(c) {
  newPositionMap.set(c, getPosition(c.el));
}
function applyTranslation(c) {
  const oldPos = positionMap.get(c);
  const newPos = newPositionMap.get(c);
  const dx = oldPos.left - newPos.left;
  const dy = oldPos.top - newPos.top;
  if (dx || dy) {
    const el = c.el;
    const s = el.style;
    const rect = el.getBoundingClientRect();
    let scaleX = 1;
    let scaleY = 1;
    if (el.offsetWidth) scaleX = rect.width / el.offsetWidth;
    if (el.offsetHeight) scaleY = rect.height / el.offsetHeight;
    if (!Number.isFinite(scaleX) || scaleX === 0) scaleX = 1;
    if (!Number.isFinite(scaleY) || scaleY === 0) scaleY = 1;
    if (Math.abs(scaleX - 1) < 0.01) scaleX = 1;
    if (Math.abs(scaleY - 1) < 0.01) scaleY = 1;
    s.transform = s.webkitTransform = `translate(${dx / scaleX}px,${dy / scaleY}px)`;
    s.transitionDuration = "0s";
    return c;
  }
}
function getPosition(el) {
  const rect = el.getBoundingClientRect();
  return {
    left: rect.left,
    top: rect.top
  };
}
function hasCSSTransform(el, root, moveClass) {
  const clone = el.cloneNode();
  const _vtc = el[vtcKey];
  if (_vtc) {
    _vtc.forEach((cls) => {
      cls.split(/\s+/).forEach((c) => c && clone.classList.remove(c));
    });
  }
  moveClass.split(/\s+/).forEach((c) => c && clone.classList.add(c));
  clone.style.display = "none";
  const container = root.nodeType === 1 ? root : root.parentNode;
  container.appendChild(clone);
  const { hasTransform } = getTransitionInfo(clone);
  container.removeChild(clone);
  return hasTransform;
}
const getModelAssigner = (vnode) => {
  const fn = vnode.props["onUpdate:modelValue"] || false;
  return isArray$1(fn) ? (value) => invokeArrayFns(fn, value) : fn;
};
function onCompositionStart(e) {
  e.target.composing = true;
}
function onCompositionEnd(e) {
  const target = e.target;
  if (target.composing) {
    target.composing = false;
    target.dispatchEvent(new Event("input"));
  }
}
const assignKey = /* @__PURE__ */ Symbol("_assign");
function castValue(value, trim, number2) {
  if (trim) value = value.trim();
  if (number2) value = looseToNumber(value);
  return value;
}
const vModelText = {
  created(el, { modifiers: { lazy, trim, number: number2 } }, vnode) {
    el[assignKey] = getModelAssigner(vnode);
    const castToNumber = number2 || vnode.props && vnode.props.type === "number";
    addEventListener(el, lazy ? "change" : "input", (e) => {
      if (e.target.composing) return;
      el[assignKey](castValue(el.value, trim, castToNumber));
    });
    if (trim || castToNumber) {
      addEventListener(el, "change", () => {
        el.value = castValue(el.value, trim, castToNumber);
      });
    }
    if (!lazy) {
      addEventListener(el, "compositionstart", onCompositionStart);
      addEventListener(el, "compositionend", onCompositionEnd);
      addEventListener(el, "change", onCompositionEnd);
    }
  },
  // set value on mounted so it's after min/max for type="range"
  mounted(el, { value }) {
    el.value = value == null ? "" : value;
  },
  beforeUpdate(el, { value, oldValue, modifiers: { lazy, trim, number: number2 } }, vnode) {
    el[assignKey] = getModelAssigner(vnode);
    if (el.composing) return;
    const elValue = (number2 || el.type === "number") && !/^0\d/.test(el.value) ? looseToNumber(el.value) : el.value;
    const newValue = value == null ? "" : value;
    if (elValue === newValue) {
      return;
    }
    const rootNode = el.getRootNode();
    if ((rootNode instanceof Document || rootNode instanceof ShadowRoot) && rootNode.activeElement === el && el.type !== "range") {
      if (lazy && value === oldValue) {
        return;
      }
      if (trim && el.value.trim() === newValue) {
        return;
      }
    }
    el.value = newValue;
  }
};
const vModelSelect = {
  // <select multiple> value need to be deep traversed
  deep: true,
  created(el, { value, modifiers: { number: number2 } }, vnode) {
    const isSetModel = isSet(value);
    addEventListener(el, "change", () => {
      const selectedVal = Array.prototype.filter.call(el.options, (o) => o.selected).map(
        (o) => number2 ? looseToNumber(getValue(o)) : getValue(o)
      );
      el[assignKey](
        el.multiple ? isSetModel ? new Set(selectedVal) : selectedVal : selectedVal[0]
      );
      el._assigning = true;
      nextTick(() => {
        el._assigning = false;
      });
    });
    el[assignKey] = getModelAssigner(vnode);
  },
  // set value in mounted & updated because <select> relies on its children
  // <option>s.
  mounted(el, { value }) {
    setSelected(el, value);
  },
  beforeUpdate(el, _binding, vnode) {
    el[assignKey] = getModelAssigner(vnode);
  },
  updated(el, { value }) {
    if (!el._assigning) {
      setSelected(el, value);
    }
  }
};
function setSelected(el, value) {
  const isMultiple = el.multiple;
  const isArrayValue = isArray$1(value);
  if (isMultiple && !isArrayValue && !isSet(value)) {
    return;
  }
  for (let i = 0, l = el.options.length; i < l; i++) {
    const option = el.options[i];
    const optionValue = getValue(option);
    if (isMultiple) {
      if (isArrayValue) {
        const optionType = typeof optionValue;
        if (optionType === "string" || optionType === "number") {
          option.selected = value.some((v) => String(v) === String(optionValue));
        } else {
          option.selected = looseIndexOf(value, optionValue) > -1;
        }
      } else {
        option.selected = value.has(optionValue);
      }
    } else if (looseEqual(getValue(option), value)) {
      if (el.selectedIndex !== i) el.selectedIndex = i;
      return;
    }
  }
  if (!isMultiple && el.selectedIndex !== -1) {
    el.selectedIndex = -1;
  }
}
function getValue(el) {
  return "_value" in el ? el._value : el.value;
}
const systemModifiers = ["ctrl", "shift", "alt", "meta"];
const modifierGuards = {
  stop: (e) => e.stopPropagation(),
  prevent: (e) => e.preventDefault(),
  self: (e) => e.target !== e.currentTarget,
  ctrl: (e) => !e.ctrlKey,
  shift: (e) => !e.shiftKey,
  alt: (e) => !e.altKey,
  meta: (e) => !e.metaKey,
  left: (e) => "button" in e && e.button !== 0,
  middle: (e) => "button" in e && e.button !== 1,
  right: (e) => "button" in e && e.button !== 2,
  exact: (e, modifiers) => systemModifiers.some((m) => e[`${m}Key`] && !modifiers.includes(m))
};
const withModifiers = (fn, modifiers) => {
  if (!fn) return fn;
  const cache2 = fn._withMods || (fn._withMods = {});
  const cacheKey = modifiers.join(".");
  return cache2[cacheKey] || (cache2[cacheKey] = (event, ...args) => {
    for (let i = 0; i < modifiers.length; i++) {
      const guard2 = modifierGuards[modifiers[i]];
      if (guard2 && guard2(event, modifiers)) return;
    }
    return fn(event, ...args);
  });
};
const keyNames = {
  esc: "escape",
  space: " ",
  up: "arrow-up",
  left: "arrow-left",
  right: "arrow-right",
  down: "arrow-down",
  delete: "backspace"
};
const withKeys = (fn, modifiers) => {
  const cache2 = fn._withKeys || (fn._withKeys = {});
  const cacheKey = modifiers.join(".");
  return cache2[cacheKey] || (cache2[cacheKey] = (event) => {
    if (!("key" in event)) {
      return;
    }
    const eventKey = hyphenate(event.key);
    if (modifiers.some(
      (k) => k === eventKey || keyNames[k] === eventKey
    )) {
      return fn(event);
    }
  });
};
const rendererOptions = /* @__PURE__ */ extend({ patchProp }, nodeOps);
let renderer;
function ensureRenderer() {
  return renderer || (renderer = createRenderer(rendererOptions));
}
const createApp = (...args) => {
  const app = ensureRenderer().createApp(...args);
  const { mount } = app;
  app.mount = (containerOrSelector) => {
    const container = normalizeContainer(containerOrSelector);
    if (!container) return;
    const component = app._component;
    if (!isFunction$1(component) && !component.render && !component.template) {
      component.template = container.innerHTML;
    }
    if (container.nodeType === 1) {
      container.textContent = "";
    }
    const proxy = mount(container, false, resolveRootNamespace(container));
    if (container instanceof Element) {
      container.removeAttribute("v-cloak");
      container.setAttribute("data-v-app", "");
    }
    return proxy;
  };
  return app;
};
function resolveRootNamespace(container) {
  if (container instanceof SVGElement) {
    return "svg";
  }
  if (typeof MathMLElement === "function" && container instanceof MathMLElement) {
    return "mathml";
  }
}
function normalizeContainer(container) {
  if (isString$2(container)) {
    const res = document.querySelector(container);
    return res;
  }
  return container;
}
/*!
 * pinia v2.3.1
 * (c) 2025 Eduardo San Martin Morote
 * @license MIT
 */
let activePinia;
const setActivePinia = (pinia) => activePinia = pinia;
const piniaSymbol = (
  /* istanbul ignore next */
  Symbol()
);
function isPlainObject$1(o) {
  return o && typeof o === "object" && Object.prototype.toString.call(o) === "[object Object]" && typeof o.toJSON !== "function";
}
var MutationType;
(function(MutationType2) {
  MutationType2["direct"] = "direct";
  MutationType2["patchObject"] = "patch object";
  MutationType2["patchFunction"] = "patch function";
})(MutationType || (MutationType = {}));
function createPinia() {
  const scope = effectScope(true);
  const state = scope.run(() => /* @__PURE__ */ ref({}));
  let _p = [];
  let toBeInstalled = [];
  const pinia = markRaw({
    install(app) {
      setActivePinia(pinia);
      {
        pinia._a = app;
        app.provide(piniaSymbol, pinia);
        app.config.globalProperties.$pinia = pinia;
        toBeInstalled.forEach((plugin) => _p.push(plugin));
        toBeInstalled = [];
      }
    },
    use(plugin) {
      if (!this._a && true) {
        toBeInstalled.push(plugin);
      } else {
        _p.push(plugin);
      }
      return this;
    },
    _p,
    // it's actually undefined here
    // @ts-expect-error
    _a: null,
    _e: scope,
    _s: /* @__PURE__ */ new Map(),
    state
  });
  return pinia;
}
const noop = () => {
};
function addSubscription(subscriptions, callback, detached, onCleanup = noop) {
  subscriptions.push(callback);
  const removeSubscription = () => {
    const idx = subscriptions.indexOf(callback);
    if (idx > -1) {
      subscriptions.splice(idx, 1);
      onCleanup();
    }
  };
  if (!detached && getCurrentScope()) {
    onScopeDispose(removeSubscription);
  }
  return removeSubscription;
}
function triggerSubscriptions(subscriptions, ...args) {
  subscriptions.slice().forEach((callback) => {
    callback(...args);
  });
}
const fallbackRunWithContext = (fn) => fn();
const ACTION_MARKER = Symbol();
const ACTION_NAME = Symbol();
function mergeReactiveObjects(target, patchToApply) {
  if (target instanceof Map && patchToApply instanceof Map) {
    patchToApply.forEach((value, key) => target.set(key, value));
  } else if (target instanceof Set && patchToApply instanceof Set) {
    patchToApply.forEach(target.add, target);
  }
  for (const key in patchToApply) {
    if (!patchToApply.hasOwnProperty(key))
      continue;
    const subPatch = patchToApply[key];
    const targetValue = target[key];
    if (isPlainObject$1(targetValue) && isPlainObject$1(subPatch) && target.hasOwnProperty(key) && !/* @__PURE__ */ isRef(subPatch) && !/* @__PURE__ */ isReactive(subPatch)) {
      target[key] = mergeReactiveObjects(targetValue, subPatch);
    } else {
      target[key] = subPatch;
    }
  }
  return target;
}
const skipHydrateSymbol = (
  /* istanbul ignore next */
  Symbol()
);
function shouldHydrate(obj) {
  return !isPlainObject$1(obj) || !obj.hasOwnProperty(skipHydrateSymbol);
}
const { assign: assign$2 } = Object;
function isComputed(o) {
  return !!(/* @__PURE__ */ isRef(o) && o.effect);
}
function createOptionsStore(id, options, pinia, hot) {
  const { state, actions, getters } = options;
  const initialState = pinia.state.value[id];
  let store;
  function setup() {
    if (!initialState && true) {
      {
        pinia.state.value[id] = state ? state() : {};
      }
    }
    const localState = /* @__PURE__ */ toRefs(pinia.state.value[id]);
    return assign$2(localState, actions, Object.keys(getters || {}).reduce((computedGetters, name) => {
      computedGetters[name] = markRaw(computed(() => {
        setActivePinia(pinia);
        const store2 = pinia._s.get(id);
        return getters[name].call(store2, store2);
      }));
      return computedGetters;
    }, {}));
  }
  store = createSetupStore(id, setup, options, pinia, hot, true);
  return store;
}
function createSetupStore($id, setup, options = {}, pinia, hot, isOptionsStore) {
  let scope;
  const optionsForPlugin = assign$2({ actions: {} }, options);
  const $subscribeOptions = { deep: true };
  let isListening;
  let isSyncListening;
  let subscriptions = [];
  let actionSubscriptions = [];
  let debuggerEvents;
  const initialState = pinia.state.value[$id];
  if (!isOptionsStore && !initialState && true) {
    {
      pinia.state.value[$id] = {};
    }
  }
  let activeListener;
  function $patch(partialStateOrMutator) {
    let subscriptionMutation;
    isListening = isSyncListening = false;
    if (typeof partialStateOrMutator === "function") {
      partialStateOrMutator(pinia.state.value[$id]);
      subscriptionMutation = {
        type: MutationType.patchFunction,
        storeId: $id,
        events: debuggerEvents
      };
    } else {
      mergeReactiveObjects(pinia.state.value[$id], partialStateOrMutator);
      subscriptionMutation = {
        type: MutationType.patchObject,
        payload: partialStateOrMutator,
        storeId: $id,
        events: debuggerEvents
      };
    }
    const myListenerId = activeListener = Symbol();
    nextTick().then(() => {
      if (activeListener === myListenerId) {
        isListening = true;
      }
    });
    isSyncListening = true;
    triggerSubscriptions(subscriptions, subscriptionMutation, pinia.state.value[$id]);
  }
  const $reset = isOptionsStore ? function $reset2() {
    const { state } = options;
    const newState = state ? state() : {};
    this.$patch(($state) => {
      assign$2($state, newState);
    });
  } : (
    /* istanbul ignore next */
    noop
  );
  function $dispose() {
    scope.stop();
    subscriptions = [];
    actionSubscriptions = [];
    pinia._s.delete($id);
  }
  const action = (fn, name = "") => {
    if (ACTION_MARKER in fn) {
      fn[ACTION_NAME] = name;
      return fn;
    }
    const wrappedAction = function() {
      setActivePinia(pinia);
      const args = Array.from(arguments);
      const afterCallbackList = [];
      const onErrorCallbackList = [];
      function after(callback) {
        afterCallbackList.push(callback);
      }
      function onError(callback) {
        onErrorCallbackList.push(callback);
      }
      triggerSubscriptions(actionSubscriptions, {
        args,
        name: wrappedAction[ACTION_NAME],
        store,
        after,
        onError
      });
      let ret;
      try {
        ret = fn.apply(this && this.$id === $id ? this : store, args);
      } catch (error) {
        triggerSubscriptions(onErrorCallbackList, error);
        throw error;
      }
      if (ret instanceof Promise) {
        return ret.then((value) => {
          triggerSubscriptions(afterCallbackList, value);
          return value;
        }).catch((error) => {
          triggerSubscriptions(onErrorCallbackList, error);
          return Promise.reject(error);
        });
      }
      triggerSubscriptions(afterCallbackList, ret);
      return ret;
    };
    wrappedAction[ACTION_MARKER] = true;
    wrappedAction[ACTION_NAME] = name;
    return wrappedAction;
  };
  const partialStore = {
    _p: pinia,
    // _s: scope,
    $id,
    $onAction: addSubscription.bind(null, actionSubscriptions),
    $patch,
    $reset,
    $subscribe(callback, options2 = {}) {
      const removeSubscription = addSubscription(subscriptions, callback, options2.detached, () => stopWatcher());
      const stopWatcher = scope.run(() => watch(() => pinia.state.value[$id], (state) => {
        if (options2.flush === "sync" ? isSyncListening : isListening) {
          callback({
            storeId: $id,
            type: MutationType.direct,
            events: debuggerEvents
          }, state);
        }
      }, assign$2({}, $subscribeOptions, options2)));
      return removeSubscription;
    },
    $dispose
  };
  const store = /* @__PURE__ */ reactive(partialStore);
  pinia._s.set($id, store);
  const runWithContext = pinia._a && pinia._a.runWithContext || fallbackRunWithContext;
  const setupStore = runWithContext(() => pinia._e.run(() => (scope = effectScope()).run(() => setup({ action }))));
  for (const key in setupStore) {
    const prop = setupStore[key];
    if (/* @__PURE__ */ isRef(prop) && !isComputed(prop) || /* @__PURE__ */ isReactive(prop)) {
      if (!isOptionsStore) {
        if (initialState && shouldHydrate(prop)) {
          if (/* @__PURE__ */ isRef(prop)) {
            prop.value = initialState[key];
          } else {
            mergeReactiveObjects(prop, initialState[key]);
          }
        }
        {
          pinia.state.value[$id][key] = prop;
        }
      }
    } else if (typeof prop === "function") {
      const actionValue = action(prop, key);
      {
        setupStore[key] = actionValue;
      }
      optionsForPlugin.actions[key] = prop;
    } else ;
  }
  {
    assign$2(store, setupStore);
    assign$2(/* @__PURE__ */ toRaw(store), setupStore);
  }
  Object.defineProperty(store, "$state", {
    get: () => pinia.state.value[$id],
    set: (state) => {
      $patch(($state) => {
        assign$2($state, state);
      });
    }
  });
  pinia._p.forEach((extender) => {
    {
      assign$2(store, scope.run(() => extender({
        store,
        app: pinia._a,
        pinia,
        options: optionsForPlugin
      })));
    }
  });
  if (initialState && isOptionsStore && options.hydrate) {
    options.hydrate(store.$state, initialState);
  }
  isListening = true;
  isSyncListening = true;
  return store;
}
/*! #__NO_SIDE_EFFECTS__ */
// @__NO_SIDE_EFFECTS__
function defineStore(idOrOptions, setup, setupOptions) {
  let id;
  let options;
  const isSetupStore = typeof setup === "function";
  if (typeof idOrOptions === "string") {
    id = idOrOptions;
    options = isSetupStore ? setupOptions : setup;
  } else {
    options = idOrOptions;
    id = idOrOptions.id;
  }
  function useStore(pinia, hot) {
    const hasContext = hasInjectionContext();
    pinia = // in test mode, ignore the argument provided as we can always retrieve a
    // pinia instance with getActivePinia()
    pinia || (hasContext ? inject(piniaSymbol, null) : null);
    if (pinia)
      setActivePinia(pinia);
    pinia = activePinia;
    if (!pinia._s.has(id)) {
      if (isSetupStore) {
        createSetupStore(id, setup, options, pinia);
      } else {
        createOptionsStore(id, options, pinia);
      }
    }
    const store = pinia._s.get(id);
    return store;
  }
  useStore.$id = id;
  return useStore;
}
function storeToRefs(store) {
  {
    const rawStore = /* @__PURE__ */ toRaw(store);
    const refs = {};
    for (const key in rawStore) {
      const value = rawStore[key];
      if (value.effect) {
        refs[key] = // ...
        computed({
          get: () => store[key],
          set(value2) {
            store[key] = value2;
          }
        });
      } else if (/* @__PURE__ */ isRef(value) || /* @__PURE__ */ isReactive(value)) {
        refs[key] = // ---
        /* @__PURE__ */ toRef(store, key);
      }
    }
    return refs;
  }
}
/*!
  * shared v9.14.5
  * (c) 2025 kazuya kawaguchi
  * Released under the MIT License.
  */
function warn(msg, err) {
  if (typeof console !== "undefined") {
    console.warn(`[intlify] ` + msg);
    if (err) {
      console.warn(err.stack);
    }
  }
}
const inBrowser = typeof window !== "undefined";
const makeSymbol = (name, shareable = false) => !shareable ? Symbol(name) : Symbol.for(name);
const generateFormatCacheKey = (locale, key, source) => friendlyJSONstringify({ l: locale, k: key, s: source });
const friendlyJSONstringify = (json) => JSON.stringify(json).replace(/\u2028/g, "\\u2028").replace(/\u2029/g, "\\u2029").replace(/\u0027/g, "\\u0027");
const isNumber = (val) => typeof val === "number" && isFinite(val);
const isDate = (val) => toTypeString(val) === "[object Date]";
const isRegExp = (val) => toTypeString(val) === "[object RegExp]";
const isEmptyObject = (val) => isPlainObject(val) && Object.keys(val).length === 0;
const assign$1 = Object.assign;
const _create = Object.create;
const create = (obj = null) => _create(obj);
let _globalThis;
const getGlobalThis = () => {
  return _globalThis || (_globalThis = typeof globalThis !== "undefined" ? globalThis : typeof self !== "undefined" ? self : typeof window !== "undefined" ? window : typeof global !== "undefined" ? global : create());
};
function escapeHtml$1(rawText) {
  return rawText.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&apos;").replace(/\//g, "&#x2F;").replace(/=/g, "&#x3D;");
}
function escapeAttributeValue(value) {
  return value.replace(/&(?![a-zA-Z0-9#]{2,6};)/g, "&amp;").replace(/"/g, "&quot;").replace(/'/g, "&apos;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}
function sanitizeTranslatedHtml(html) {
  html = html.replace(/(\w+)\s*=\s*"([^"]*)"/g, (_, attrName, attrValue) => `${attrName}="${escapeAttributeValue(attrValue)}"`);
  html = html.replace(/(\w+)\s*=\s*'([^']*)'/g, (_, attrName, attrValue) => `${attrName}='${escapeAttributeValue(attrValue)}'`);
  const eventHandlerPattern = /\s*on\w+\s*=\s*["']?[^"'>]+["']?/gi;
  if (eventHandlerPattern.test(html)) {
    html = html.replace(/(\s+)(on)(\w+\s*=)/gi, "$1&#111;n$3");
  }
  const javascriptUrlPattern = [
    // In href, src, action, formaction attributes
    /(\s+(?:href|src|action|formaction)\s*=\s*["']?)\s*javascript:/gi,
    // In style attributes within url()
    /(style\s*=\s*["'][^"']*url\s*\(\s*)javascript:/gi
  ];
  javascriptUrlPattern.forEach((pattern) => {
    html = html.replace(pattern, "$1javascript&#58;");
  });
  return html;
}
const hasOwnProperty = Object.prototype.hasOwnProperty;
function hasOwn(obj, key) {
  return hasOwnProperty.call(obj, key);
}
const isArray = Array.isArray;
const isFunction = (val) => typeof val === "function";
const isString$1 = (val) => typeof val === "string";
const isBoolean = (val) => typeof val === "boolean";
const isObject$1 = (val) => val !== null && typeof val === "object";
const isPromise = (val) => {
  return isObject$1(val) && isFunction(val.then) && isFunction(val.catch);
};
const objectToString = Object.prototype.toString;
const toTypeString = (value) => objectToString.call(value);
const isPlainObject = (val) => {
  if (!isObject$1(val))
    return false;
  const proto = Object.getPrototypeOf(val);
  return proto === null || proto.constructor === Object;
};
const toDisplayString = (val) => {
  return val == null ? "" : isArray(val) || isPlainObject(val) && val.toString === objectToString ? JSON.stringify(val, null, 2) : String(val);
};
function join$1(items, separator2 = "") {
  return items.reduce((str, item, index) => index === 0 ? str + item : str + separator2 + item, "");
}
function incrementer(code2) {
  let current = code2;
  return () => ++current;
}
const isNotObjectOrIsArray = (val) => !isObject$1(val) || isArray(val);
function deepCopy(src, des) {
  if (isNotObjectOrIsArray(src) || isNotObjectOrIsArray(des)) {
    throw new Error("Invalid value");
  }
  const stack2 = [{ src, des }];
  while (stack2.length) {
    const { src: src2, des: des2 } = stack2.pop();
    Object.keys(src2).forEach((key) => {
      if (key === "__proto__") {
        return;
      }
      if (isObject$1(src2[key]) && !isObject$1(des2[key])) {
        des2[key] = Array.isArray(src2[key]) ? [] : create();
      }
      if (isNotObjectOrIsArray(des2[key]) || isNotObjectOrIsArray(src2[key])) {
        des2[key] = src2[key];
      } else {
        stack2.push({ src: src2[key], des: des2[key] });
      }
    });
  }
}
/*!
  * message-compiler v9.14.5
  * (c) 2025 kazuya kawaguchi
  * Released under the MIT License.
  */
function createPosition(line, column, offset) {
  return { line, column, offset };
}
function createLocation(start, end, source) {
  const loc = { start, end };
  return loc;
}
const RE_ARGS = /\{([0-9a-zA-Z]+)\}/g;
function format$1(message, ...args) {
  if (args.length === 1 && isObject(args[0])) {
    args = args[0];
  }
  if (!args || !args.hasOwnProperty) {
    args = {};
  }
  return message.replace(RE_ARGS, (match, identifier) => {
    return args.hasOwnProperty(identifier) ? args[identifier] : "";
  });
}
const assign = Object.assign;
const isString = (val) => typeof val === "string";
const isObject = (val) => val !== null && typeof val === "object";
function join(items, separator2 = "") {
  return items.reduce((str, item, index) => index === 0 ? str + item : str + separator2 + item, "");
}
const CompileWarnCodes = {
  USE_MODULO_SYNTAX: 1,
  __EXTEND_POINT__: 2
};
const warnMessages = {
  [CompileWarnCodes.USE_MODULO_SYNTAX]: `Use modulo before '{{0}}'.`
};
function createCompileWarn(code2, loc, ...args) {
  const msg = format$1(warnMessages[code2], ...args || []);
  const message = { message: String(msg), code: code2 };
  if (loc) {
    message.location = loc;
  }
  return message;
}
const CompileErrorCodes = {
  // tokenizer error codes
  EXPECTED_TOKEN: 1,
  INVALID_TOKEN_IN_PLACEHOLDER: 2,
  UNTERMINATED_SINGLE_QUOTE_IN_PLACEHOLDER: 3,
  UNKNOWN_ESCAPE_SEQUENCE: 4,
  INVALID_UNICODE_ESCAPE_SEQUENCE: 5,
  UNBALANCED_CLOSING_BRACE: 6,
  UNTERMINATED_CLOSING_BRACE: 7,
  EMPTY_PLACEHOLDER: 8,
  NOT_ALLOW_NEST_PLACEHOLDER: 9,
  INVALID_LINKED_FORMAT: 10,
  // parser error codes
  MUST_HAVE_MESSAGES_IN_PLURAL: 11,
  UNEXPECTED_EMPTY_LINKED_MODIFIER: 12,
  UNEXPECTED_EMPTY_LINKED_KEY: 13,
  UNEXPECTED_LEXICAL_ANALYSIS: 14,
  // generator error codes
  UNHANDLED_CODEGEN_NODE_TYPE: 15,
  // minifier error codes
  UNHANDLED_MINIFIER_NODE_TYPE: 16,
  // Special value for higher-order compilers to pick up the last code
  // to avoid collision of error codes. This should always be kept as the last
  // item.
  __EXTEND_POINT__: 17
};
const errorMessages = {
  // tokenizer error messages
  [CompileErrorCodes.EXPECTED_TOKEN]: `Expected token: '{0}'`,
  [CompileErrorCodes.INVALID_TOKEN_IN_PLACEHOLDER]: `Invalid token in placeholder: '{0}'`,
  [CompileErrorCodes.UNTERMINATED_SINGLE_QUOTE_IN_PLACEHOLDER]: `Unterminated single quote in placeholder`,
  [CompileErrorCodes.UNKNOWN_ESCAPE_SEQUENCE]: `Unknown escape sequence: \\{0}`,
  [CompileErrorCodes.INVALID_UNICODE_ESCAPE_SEQUENCE]: `Invalid unicode escape sequence: {0}`,
  [CompileErrorCodes.UNBALANCED_CLOSING_BRACE]: `Unbalanced closing brace`,
  [CompileErrorCodes.UNTERMINATED_CLOSING_BRACE]: `Unterminated closing brace`,
  [CompileErrorCodes.EMPTY_PLACEHOLDER]: `Empty placeholder`,
  [CompileErrorCodes.NOT_ALLOW_NEST_PLACEHOLDER]: `Not allowed nest placeholder`,
  [CompileErrorCodes.INVALID_LINKED_FORMAT]: `Invalid linked format`,
  // parser error messages
  [CompileErrorCodes.MUST_HAVE_MESSAGES_IN_PLURAL]: `Plural must have messages`,
  [CompileErrorCodes.UNEXPECTED_EMPTY_LINKED_MODIFIER]: `Unexpected empty linked modifier`,
  [CompileErrorCodes.UNEXPECTED_EMPTY_LINKED_KEY]: `Unexpected empty linked key`,
  [CompileErrorCodes.UNEXPECTED_LEXICAL_ANALYSIS]: `Unexpected lexical analysis in token: '{0}'`,
  // generator error messages
  [CompileErrorCodes.UNHANDLED_CODEGEN_NODE_TYPE]: `unhandled codegen node type: '{0}'`,
  // minimizer error messages
  [CompileErrorCodes.UNHANDLED_MINIFIER_NODE_TYPE]: `unhandled mimifier node type: '{0}'`
};
function createCompileError(code2, loc, options = {}) {
  const { domain, messages, args } = options;
  const msg = format$1((messages || errorMessages)[code2] || "", ...args || []);
  const error = new SyntaxError(String(msg));
  error.code = code2;
  if (loc) {
    error.location = loc;
  }
  error.domain = domain;
  return error;
}
function defaultOnError(error) {
  throw error;
}
const CHAR_SP = " ";
const CHAR_CR = "\r";
const CHAR_LF = "\n";
const CHAR_LS = String.fromCharCode(8232);
const CHAR_PS = String.fromCharCode(8233);
function createScanner(str) {
  const _buf = str;
  let _index = 0;
  let _line = 1;
  let _column = 1;
  let _peekOffset = 0;
  const isCRLF = (index2) => _buf[index2] === CHAR_CR && _buf[index2 + 1] === CHAR_LF;
  const isLF = (index2) => _buf[index2] === CHAR_LF;
  const isPS = (index2) => _buf[index2] === CHAR_PS;
  const isLS = (index2) => _buf[index2] === CHAR_LS;
  const isLineEnd = (index2) => isCRLF(index2) || isLF(index2) || isPS(index2) || isLS(index2);
  const index = () => _index;
  const line = () => _line;
  const column = () => _column;
  const peekOffset = () => _peekOffset;
  const charAt = (offset) => isCRLF(offset) || isPS(offset) || isLS(offset) ? CHAR_LF : _buf[offset];
  const currentChar = () => charAt(_index);
  const currentPeek = () => charAt(_index + _peekOffset);
  function next() {
    _peekOffset = 0;
    if (isLineEnd(_index)) {
      _line++;
      _column = 0;
    }
    if (isCRLF(_index)) {
      _index++;
    }
    _index++;
    _column++;
    return _buf[_index];
  }
  function peek() {
    if (isCRLF(_index + _peekOffset)) {
      _peekOffset++;
    }
    _peekOffset++;
    return _buf[_index + _peekOffset];
  }
  function reset() {
    _index = 0;
    _line = 1;
    _column = 1;
    _peekOffset = 0;
  }
  function resetPeek(offset = 0) {
    _peekOffset = offset;
  }
  function skipToPeek() {
    const target = _index + _peekOffset;
    while (target !== _index) {
      next();
    }
    _peekOffset = 0;
  }
  return {
    index,
    line,
    column,
    peekOffset,
    charAt,
    currentChar,
    currentPeek,
    next,
    peek,
    reset,
    resetPeek,
    skipToPeek
  };
}
const EOF = void 0;
const DOT = ".";
const LITERAL_DELIMITER = "'";
const ERROR_DOMAIN$3 = "tokenizer";
function createTokenizer(source, options = {}) {
  const location2 = options.location !== false;
  const _scnr = createScanner(source);
  const currentOffset = () => _scnr.index();
  const currentPosition = () => createPosition(_scnr.line(), _scnr.column(), _scnr.index());
  const _initLoc = currentPosition();
  const _initOffset = currentOffset();
  const _context = {
    currentType: 14,
    offset: _initOffset,
    startLoc: _initLoc,
    endLoc: _initLoc,
    lastType: 14,
    lastOffset: _initOffset,
    lastStartLoc: _initLoc,
    lastEndLoc: _initLoc,
    braceNest: 0,
    inLinked: false,
    text: ""
  };
  const context = () => _context;
  const { onError } = options;
  function emitError(code2, pos, offset, ...args) {
    const ctx = context();
    pos.column += offset;
    pos.offset += offset;
    if (onError) {
      const loc = location2 ? createLocation(ctx.startLoc, pos) : null;
      const err = createCompileError(code2, loc, {
        domain: ERROR_DOMAIN$3,
        args
      });
      onError(err);
    }
  }
  function getToken(context2, type, value) {
    context2.endLoc = currentPosition();
    context2.currentType = type;
    const token = { type };
    if (location2) {
      token.loc = createLocation(context2.startLoc, context2.endLoc);
    }
    if (value != null) {
      token.value = value;
    }
    return token;
  }
  const getEndToken = (context2) => getToken(
    context2,
    14
    /* TokenTypes.EOF */
  );
  function eat(scnr, ch) {
    if (scnr.currentChar() === ch) {
      scnr.next();
      return ch;
    } else {
      emitError(CompileErrorCodes.EXPECTED_TOKEN, currentPosition(), 0, ch);
      return "";
    }
  }
  function peekSpaces(scnr) {
    let buf = "";
    while (scnr.currentPeek() === CHAR_SP || scnr.currentPeek() === CHAR_LF) {
      buf += scnr.currentPeek();
      scnr.peek();
    }
    return buf;
  }
  function skipSpaces(scnr) {
    const buf = peekSpaces(scnr);
    scnr.skipToPeek();
    return buf;
  }
  function isIdentifierStart(ch) {
    if (ch === EOF) {
      return false;
    }
    const cc = ch.charCodeAt(0);
    return cc >= 97 && cc <= 122 || // a-z
    cc >= 65 && cc <= 90 || // A-Z
    cc === 95;
  }
  function isNumberStart(ch) {
    if (ch === EOF) {
      return false;
    }
    const cc = ch.charCodeAt(0);
    return cc >= 48 && cc <= 57;
  }
  function isNamedIdentifierStart(scnr, context2) {
    const { currentType } = context2;
    if (currentType !== 2) {
      return false;
    }
    peekSpaces(scnr);
    const ret = isIdentifierStart(scnr.currentPeek());
    scnr.resetPeek();
    return ret;
  }
  function isListIdentifierStart(scnr, context2) {
    const { currentType } = context2;
    if (currentType !== 2) {
      return false;
    }
    peekSpaces(scnr);
    const ch = scnr.currentPeek() === "-" ? scnr.peek() : scnr.currentPeek();
    const ret = isNumberStart(ch);
    scnr.resetPeek();
    return ret;
  }
  function isLiteralStart(scnr, context2) {
    const { currentType } = context2;
    if (currentType !== 2) {
      return false;
    }
    peekSpaces(scnr);
    const ret = scnr.currentPeek() === LITERAL_DELIMITER;
    scnr.resetPeek();
    return ret;
  }
  function isLinkedDotStart(scnr, context2) {
    const { currentType } = context2;
    if (currentType !== 8) {
      return false;
    }
    peekSpaces(scnr);
    const ret = scnr.currentPeek() === ".";
    scnr.resetPeek();
    return ret;
  }
  function isLinkedModifierStart(scnr, context2) {
    const { currentType } = context2;
    if (currentType !== 9) {
      return false;
    }
    peekSpaces(scnr);
    const ret = isIdentifierStart(scnr.currentPeek());
    scnr.resetPeek();
    return ret;
  }
  function isLinkedDelimiterStart(scnr, context2) {
    const { currentType } = context2;
    if (!(currentType === 8 || currentType === 12)) {
      return false;
    }
    peekSpaces(scnr);
    const ret = scnr.currentPeek() === ":";
    scnr.resetPeek();
    return ret;
  }
  function isLinkedReferStart(scnr, context2) {
    const { currentType } = context2;
    if (currentType !== 10) {
      return false;
    }
    const fn = () => {
      const ch = scnr.currentPeek();
      if (ch === "{") {
        return isIdentifierStart(scnr.peek());
      } else if (ch === "@" || ch === "%" || ch === "|" || ch === ":" || ch === "." || ch === CHAR_SP || !ch) {
        return false;
      } else if (ch === CHAR_LF) {
        scnr.peek();
        return fn();
      } else {
        return isTextStart(scnr, false);
      }
    };
    const ret = fn();
    scnr.resetPeek();
    return ret;
  }
  function isPluralStart(scnr) {
    peekSpaces(scnr);
    const ret = scnr.currentPeek() === "|";
    scnr.resetPeek();
    return ret;
  }
  function detectModuloStart(scnr) {
    const spaces = peekSpaces(scnr);
    const ret = scnr.currentPeek() === "%" && scnr.peek() === "{";
    scnr.resetPeek();
    return {
      isModulo: ret,
      hasSpace: spaces.length > 0
    };
  }
  function isTextStart(scnr, reset = true) {
    const fn = (hasSpace = false, prev = "", detectModulo = false) => {
      const ch = scnr.currentPeek();
      if (ch === "{") {
        return prev === "%" ? false : hasSpace;
      } else if (ch === "@" || !ch) {
        return prev === "%" ? true : hasSpace;
      } else if (ch === "%") {
        scnr.peek();
        return fn(hasSpace, "%", true);
      } else if (ch === "|") {
        return prev === "%" || detectModulo ? true : !(prev === CHAR_SP || prev === CHAR_LF);
      } else if (ch === CHAR_SP) {
        scnr.peek();
        return fn(true, CHAR_SP, detectModulo);
      } else if (ch === CHAR_LF) {
        scnr.peek();
        return fn(true, CHAR_LF, detectModulo);
      } else {
        return true;
      }
    };
    const ret = fn();
    reset && scnr.resetPeek();
    return ret;
  }
  function takeChar(scnr, fn) {
    const ch = scnr.currentChar();
    if (ch === EOF) {
      return EOF;
    }
    if (fn(ch)) {
      scnr.next();
      return ch;
    }
    return null;
  }
  function isIdentifier(ch) {
    const cc = ch.charCodeAt(0);
    return cc >= 97 && cc <= 122 || // a-z
    cc >= 65 && cc <= 90 || // A-Z
    cc >= 48 && cc <= 57 || // 0-9
    cc === 95 || // _
    cc === 36;
  }
  function takeIdentifierChar(scnr) {
    return takeChar(scnr, isIdentifier);
  }
  function isNamedIdentifier(ch) {
    const cc = ch.charCodeAt(0);
    return cc >= 97 && cc <= 122 || // a-z
    cc >= 65 && cc <= 90 || // A-Z
    cc >= 48 && cc <= 57 || // 0-9
    cc === 95 || // _
    cc === 36 || // $
    cc === 45;
  }
  function takeNamedIdentifierChar(scnr) {
    return takeChar(scnr, isNamedIdentifier);
  }
  function isDigit(ch) {
    const cc = ch.charCodeAt(0);
    return cc >= 48 && cc <= 57;
  }
  function takeDigit(scnr) {
    return takeChar(scnr, isDigit);
  }
  function isHexDigit(ch) {
    const cc = ch.charCodeAt(0);
    return cc >= 48 && cc <= 57 || // 0-9
    cc >= 65 && cc <= 70 || // A-F
    cc >= 97 && cc <= 102;
  }
  function takeHexDigit(scnr) {
    return takeChar(scnr, isHexDigit);
  }
  function getDigits(scnr) {
    let ch = "";
    let num = "";
    while (ch = takeDigit(scnr)) {
      num += ch;
    }
    return num;
  }
  function readModulo(scnr) {
    skipSpaces(scnr);
    const ch = scnr.currentChar();
    if (ch !== "%") {
      emitError(CompileErrorCodes.EXPECTED_TOKEN, currentPosition(), 0, ch);
    }
    scnr.next();
    return "%";
  }
  function readText(scnr) {
    let buf = "";
    while (true) {
      const ch = scnr.currentChar();
      if (ch === "{" || ch === "}" || ch === "@" || ch === "|" || !ch) {
        break;
      } else if (ch === "%") {
        if (isTextStart(scnr)) {
          buf += ch;
          scnr.next();
        } else {
          break;
        }
      } else if (ch === CHAR_SP || ch === CHAR_LF) {
        if (isTextStart(scnr)) {
          buf += ch;
          scnr.next();
        } else if (isPluralStart(scnr)) {
          break;
        } else {
          buf += ch;
          scnr.next();
        }
      } else {
        buf += ch;
        scnr.next();
      }
    }
    return buf;
  }
  function readNamedIdentifier(scnr) {
    skipSpaces(scnr);
    let ch = "";
    let name = "";
    while (ch = takeNamedIdentifierChar(scnr)) {
      name += ch;
    }
    if (scnr.currentChar() === EOF) {
      emitError(CompileErrorCodes.UNTERMINATED_CLOSING_BRACE, currentPosition(), 0);
    }
    return name;
  }
  function readListIdentifier(scnr) {
    skipSpaces(scnr);
    let value = "";
    if (scnr.currentChar() === "-") {
      scnr.next();
      value += `-${getDigits(scnr)}`;
    } else {
      value += getDigits(scnr);
    }
    if (scnr.currentChar() === EOF) {
      emitError(CompileErrorCodes.UNTERMINATED_CLOSING_BRACE, currentPosition(), 0);
    }
    return value;
  }
  function isLiteral2(ch) {
    return ch !== LITERAL_DELIMITER && ch !== CHAR_LF;
  }
  function readLiteral(scnr) {
    skipSpaces(scnr);
    eat(scnr, `'`);
    let ch = "";
    let literal = "";
    while (ch = takeChar(scnr, isLiteral2)) {
      if (ch === "\\") {
        literal += readEscapeSequence(scnr);
      } else {
        literal += ch;
      }
    }
    const current = scnr.currentChar();
    if (current === CHAR_LF || current === EOF) {
      emitError(CompileErrorCodes.UNTERMINATED_SINGLE_QUOTE_IN_PLACEHOLDER, currentPosition(), 0);
      if (current === CHAR_LF) {
        scnr.next();
        eat(scnr, `'`);
      }
      return literal;
    }
    eat(scnr, `'`);
    return literal;
  }
  function readEscapeSequence(scnr) {
    const ch = scnr.currentChar();
    switch (ch) {
      case "\\":
      case `'`:
        scnr.next();
        return `\\${ch}`;
      case "u":
        return readUnicodeEscapeSequence(scnr, ch, 4);
      case "U":
        return readUnicodeEscapeSequence(scnr, ch, 6);
      default:
        emitError(CompileErrorCodes.UNKNOWN_ESCAPE_SEQUENCE, currentPosition(), 0, ch);
        return "";
    }
  }
  function readUnicodeEscapeSequence(scnr, unicode, digits) {
    eat(scnr, unicode);
    let sequence = "";
    for (let i = 0; i < digits; i++) {
      const ch = takeHexDigit(scnr);
      if (!ch) {
        emitError(CompileErrorCodes.INVALID_UNICODE_ESCAPE_SEQUENCE, currentPosition(), 0, `\\${unicode}${sequence}${scnr.currentChar()}`);
        break;
      }
      sequence += ch;
    }
    return `\\${unicode}${sequence}`;
  }
  function isInvalidIdentifier(ch) {
    return ch !== "{" && ch !== "}" && ch !== CHAR_SP && ch !== CHAR_LF;
  }
  function readInvalidIdentifier(scnr) {
    skipSpaces(scnr);
    let ch = "";
    let identifiers = "";
    while (ch = takeChar(scnr, isInvalidIdentifier)) {
      identifiers += ch;
    }
    return identifiers;
  }
  function readLinkedModifier(scnr) {
    let ch = "";
    let name = "";
    while (ch = takeIdentifierChar(scnr)) {
      name += ch;
    }
    return name;
  }
  function readLinkedRefer(scnr) {
    const fn = (buf) => {
      const ch = scnr.currentChar();
      if (ch === "{" || ch === "%" || ch === "@" || ch === "|" || ch === "(" || ch === ")" || !ch) {
        return buf;
      } else if (ch === CHAR_SP) {
        return buf;
      } else if (ch === CHAR_LF || ch === DOT) {
        buf += ch;
        scnr.next();
        return fn(buf);
      } else {
        buf += ch;
        scnr.next();
        return fn(buf);
      }
    };
    return fn("");
  }
  function readPlural(scnr) {
    skipSpaces(scnr);
    const plural = eat(
      scnr,
      "|"
      /* TokenChars.Pipe */
    );
    skipSpaces(scnr);
    return plural;
  }
  function readTokenInPlaceholder(scnr, context2) {
    let token = null;
    const ch = scnr.currentChar();
    switch (ch) {
      case "{":
        if (context2.braceNest >= 1) {
          emitError(CompileErrorCodes.NOT_ALLOW_NEST_PLACEHOLDER, currentPosition(), 0);
        }
        scnr.next();
        token = getToken(
          context2,
          2,
          "{"
          /* TokenChars.BraceLeft */
        );
        skipSpaces(scnr);
        context2.braceNest++;
        return token;
      case "}":
        if (context2.braceNest > 0 && context2.currentType === 2) {
          emitError(CompileErrorCodes.EMPTY_PLACEHOLDER, currentPosition(), 0);
        }
        scnr.next();
        token = getToken(
          context2,
          3,
          "}"
          /* TokenChars.BraceRight */
        );
        context2.braceNest--;
        context2.braceNest > 0 && skipSpaces(scnr);
        if (context2.inLinked && context2.braceNest === 0) {
          context2.inLinked = false;
        }
        return token;
      case "@":
        if (context2.braceNest > 0) {
          emitError(CompileErrorCodes.UNTERMINATED_CLOSING_BRACE, currentPosition(), 0);
        }
        token = readTokenInLinked(scnr, context2) || getEndToken(context2);
        context2.braceNest = 0;
        return token;
      default: {
        let validNamedIdentifier = true;
        let validListIdentifier = true;
        let validLiteral = true;
        if (isPluralStart(scnr)) {
          if (context2.braceNest > 0) {
            emitError(CompileErrorCodes.UNTERMINATED_CLOSING_BRACE, currentPosition(), 0);
          }
          token = getToken(context2, 1, readPlural(scnr));
          context2.braceNest = 0;
          context2.inLinked = false;
          return token;
        }
        if (context2.braceNest > 0 && (context2.currentType === 5 || context2.currentType === 6 || context2.currentType === 7)) {
          emitError(CompileErrorCodes.UNTERMINATED_CLOSING_BRACE, currentPosition(), 0);
          context2.braceNest = 0;
          return readToken(scnr, context2);
        }
        if (validNamedIdentifier = isNamedIdentifierStart(scnr, context2)) {
          token = getToken(context2, 5, readNamedIdentifier(scnr));
          skipSpaces(scnr);
          return token;
        }
        if (validListIdentifier = isListIdentifierStart(scnr, context2)) {
          token = getToken(context2, 6, readListIdentifier(scnr));
          skipSpaces(scnr);
          return token;
        }
        if (validLiteral = isLiteralStart(scnr, context2)) {
          token = getToken(context2, 7, readLiteral(scnr));
          skipSpaces(scnr);
          return token;
        }
        if (!validNamedIdentifier && !validListIdentifier && !validLiteral) {
          token = getToken(context2, 13, readInvalidIdentifier(scnr));
          emitError(CompileErrorCodes.INVALID_TOKEN_IN_PLACEHOLDER, currentPosition(), 0, token.value);
          skipSpaces(scnr);
          return token;
        }
        break;
      }
    }
    return token;
  }
  function readTokenInLinked(scnr, context2) {
    const { currentType } = context2;
    let token = null;
    const ch = scnr.currentChar();
    if ((currentType === 8 || currentType === 9 || currentType === 12 || currentType === 10) && (ch === CHAR_LF || ch === CHAR_SP)) {
      emitError(CompileErrorCodes.INVALID_LINKED_FORMAT, currentPosition(), 0);
    }
    switch (ch) {
      case "@":
        scnr.next();
        token = getToken(
          context2,
          8,
          "@"
          /* TokenChars.LinkedAlias */
        );
        context2.inLinked = true;
        return token;
      case ".":
        skipSpaces(scnr);
        scnr.next();
        return getToken(
          context2,
          9,
          "."
          /* TokenChars.LinkedDot */
        );
      case ":":
        skipSpaces(scnr);
        scnr.next();
        return getToken(
          context2,
          10,
          ":"
          /* TokenChars.LinkedDelimiter */
        );
      default:
        if (isPluralStart(scnr)) {
          token = getToken(context2, 1, readPlural(scnr));
          context2.braceNest = 0;
          context2.inLinked = false;
          return token;
        }
        if (isLinkedDotStart(scnr, context2) || isLinkedDelimiterStart(scnr, context2)) {
          skipSpaces(scnr);
          return readTokenInLinked(scnr, context2);
        }
        if (isLinkedModifierStart(scnr, context2)) {
          skipSpaces(scnr);
          return getToken(context2, 12, readLinkedModifier(scnr));
        }
        if (isLinkedReferStart(scnr, context2)) {
          skipSpaces(scnr);
          if (ch === "{") {
            return readTokenInPlaceholder(scnr, context2) || token;
          } else {
            return getToken(context2, 11, readLinkedRefer(scnr));
          }
        }
        if (currentType === 8) {
          emitError(CompileErrorCodes.INVALID_LINKED_FORMAT, currentPosition(), 0);
        }
        context2.braceNest = 0;
        context2.inLinked = false;
        return readToken(scnr, context2);
    }
  }
  function readToken(scnr, context2) {
    let token = {
      type: 14
      /* TokenTypes.EOF */
    };
    if (context2.braceNest > 0) {
      return readTokenInPlaceholder(scnr, context2) || getEndToken(context2);
    }
    if (context2.inLinked) {
      return readTokenInLinked(scnr, context2) || getEndToken(context2);
    }
    const ch = scnr.currentChar();
    switch (ch) {
      case "{":
        return readTokenInPlaceholder(scnr, context2) || getEndToken(context2);
      case "}":
        emitError(CompileErrorCodes.UNBALANCED_CLOSING_BRACE, currentPosition(), 0);
        scnr.next();
        return getToken(
          context2,
          3,
          "}"
          /* TokenChars.BraceRight */
        );
      case "@":
        return readTokenInLinked(scnr, context2) || getEndToken(context2);
      default: {
        if (isPluralStart(scnr)) {
          token = getToken(context2, 1, readPlural(scnr));
          context2.braceNest = 0;
          context2.inLinked = false;
          return token;
        }
        const { isModulo, hasSpace } = detectModuloStart(scnr);
        if (isModulo) {
          return hasSpace ? getToken(context2, 0, readText(scnr)) : getToken(context2, 4, readModulo(scnr));
        }
        if (isTextStart(scnr)) {
          return getToken(context2, 0, readText(scnr));
        }
        break;
      }
    }
    return token;
  }
  function nextToken() {
    const { currentType, offset, startLoc, endLoc } = _context;
    _context.lastType = currentType;
    _context.lastOffset = offset;
    _context.lastStartLoc = startLoc;
    _context.lastEndLoc = endLoc;
    _context.offset = currentOffset();
    _context.startLoc = currentPosition();
    if (_scnr.currentChar() === EOF) {
      return getToken(
        _context,
        14
        /* TokenTypes.EOF */
      );
    }
    return readToken(_scnr, _context);
  }
  return {
    nextToken,
    currentOffset,
    currentPosition,
    context
  };
}
const ERROR_DOMAIN$2 = "parser";
const KNOWN_ESCAPES = /(?:\\\\|\\'|\\u([0-9a-fA-F]{4})|\\U([0-9a-fA-F]{6}))/g;
function fromEscapeSequence(match, codePoint4, codePoint6) {
  switch (match) {
    case `\\\\`:
      return `\\`;
    case `\\'`:
      return `'`;
    default: {
      const codePoint = parseInt(codePoint4 || codePoint6, 16);
      if (codePoint <= 55295 || codePoint >= 57344) {
        return String.fromCodePoint(codePoint);
      }
      return "�";
    }
  }
}
function createParser(options = {}) {
  const location2 = options.location !== false;
  const { onError, onWarn } = options;
  function emitError(tokenzer, code2, start, offset, ...args) {
    const end = tokenzer.currentPosition();
    end.offset += offset;
    end.column += offset;
    if (onError) {
      const loc = location2 ? createLocation(start, end) : null;
      const err = createCompileError(code2, loc, {
        domain: ERROR_DOMAIN$2,
        args
      });
      onError(err);
    }
  }
  function emitWarn(tokenzer, code2, start, offset, ...args) {
    const end = tokenzer.currentPosition();
    end.offset += offset;
    end.column += offset;
    if (onWarn) {
      const loc = location2 ? createLocation(start, end) : null;
      onWarn(createCompileWarn(code2, loc, args));
    }
  }
  function startNode(type, offset, loc) {
    const node = { type };
    if (location2) {
      node.start = offset;
      node.end = offset;
      node.loc = { start: loc, end: loc };
    }
    return node;
  }
  function endNode(node, offset, pos, type) {
    if (location2) {
      node.end = offset;
      if (node.loc) {
        node.loc.end = pos;
      }
    }
  }
  function parseText(tokenizer, value) {
    const context = tokenizer.context();
    const node = startNode(3, context.offset, context.startLoc);
    node.value = value;
    endNode(node, tokenizer.currentOffset(), tokenizer.currentPosition());
    return node;
  }
  function parseList(tokenizer, index) {
    const context = tokenizer.context();
    const { lastOffset: offset, lastStartLoc: loc } = context;
    const node = startNode(5, offset, loc);
    node.index = parseInt(index, 10);
    tokenizer.nextToken();
    endNode(node, tokenizer.currentOffset(), tokenizer.currentPosition());
    return node;
  }
  function parseNamed(tokenizer, key, modulo) {
    const context = tokenizer.context();
    const { lastOffset: offset, lastStartLoc: loc } = context;
    const node = startNode(4, offset, loc);
    node.key = key;
    if (modulo === true) {
      node.modulo = true;
    }
    tokenizer.nextToken();
    endNode(node, tokenizer.currentOffset(), tokenizer.currentPosition());
    return node;
  }
  function parseLiteral(tokenizer, value) {
    const context = tokenizer.context();
    const { lastOffset: offset, lastStartLoc: loc } = context;
    const node = startNode(9, offset, loc);
    node.value = value.replace(KNOWN_ESCAPES, fromEscapeSequence);
    tokenizer.nextToken();
    endNode(node, tokenizer.currentOffset(), tokenizer.currentPosition());
    return node;
  }
  function parseLinkedModifier(tokenizer) {
    const token = tokenizer.nextToken();
    const context = tokenizer.context();
    const { lastOffset: offset, lastStartLoc: loc } = context;
    const node = startNode(8, offset, loc);
    if (token.type !== 12) {
      emitError(tokenizer, CompileErrorCodes.UNEXPECTED_EMPTY_LINKED_MODIFIER, context.lastStartLoc, 0);
      node.value = "";
      endNode(node, offset, loc);
      return {
        nextConsumeToken: token,
        node
      };
    }
    if (token.value == null) {
      emitError(tokenizer, CompileErrorCodes.UNEXPECTED_LEXICAL_ANALYSIS, context.lastStartLoc, 0, getTokenCaption(token));
    }
    node.value = token.value || "";
    endNode(node, tokenizer.currentOffset(), tokenizer.currentPosition());
    return {
      node
    };
  }
  function parseLinkedKey(tokenizer, value) {
    const context = tokenizer.context();
    const node = startNode(7, context.offset, context.startLoc);
    node.value = value;
    endNode(node, tokenizer.currentOffset(), tokenizer.currentPosition());
    return node;
  }
  function parseLinked(tokenizer) {
    const context = tokenizer.context();
    const linkedNode = startNode(6, context.offset, context.startLoc);
    let token = tokenizer.nextToken();
    if (token.type === 9) {
      const parsed = parseLinkedModifier(tokenizer);
      linkedNode.modifier = parsed.node;
      token = parsed.nextConsumeToken || tokenizer.nextToken();
    }
    if (token.type !== 10) {
      emitError(tokenizer, CompileErrorCodes.UNEXPECTED_LEXICAL_ANALYSIS, context.lastStartLoc, 0, getTokenCaption(token));
    }
    token = tokenizer.nextToken();
    if (token.type === 2) {
      token = tokenizer.nextToken();
    }
    switch (token.type) {
      case 11:
        if (token.value == null) {
          emitError(tokenizer, CompileErrorCodes.UNEXPECTED_LEXICAL_ANALYSIS, context.lastStartLoc, 0, getTokenCaption(token));
        }
        linkedNode.key = parseLinkedKey(tokenizer, token.value || "");
        break;
      case 5:
        if (token.value == null) {
          emitError(tokenizer, CompileErrorCodes.UNEXPECTED_LEXICAL_ANALYSIS, context.lastStartLoc, 0, getTokenCaption(token));
        }
        linkedNode.key = parseNamed(tokenizer, token.value || "");
        break;
      case 6:
        if (token.value == null) {
          emitError(tokenizer, CompileErrorCodes.UNEXPECTED_LEXICAL_ANALYSIS, context.lastStartLoc, 0, getTokenCaption(token));
        }
        linkedNode.key = parseList(tokenizer, token.value || "");
        break;
      case 7:
        if (token.value == null) {
          emitError(tokenizer, CompileErrorCodes.UNEXPECTED_LEXICAL_ANALYSIS, context.lastStartLoc, 0, getTokenCaption(token));
        }
        linkedNode.key = parseLiteral(tokenizer, token.value || "");
        break;
      default: {
        emitError(tokenizer, CompileErrorCodes.UNEXPECTED_EMPTY_LINKED_KEY, context.lastStartLoc, 0);
        const nextContext = tokenizer.context();
        const emptyLinkedKeyNode = startNode(7, nextContext.offset, nextContext.startLoc);
        emptyLinkedKeyNode.value = "";
        endNode(emptyLinkedKeyNode, nextContext.offset, nextContext.startLoc);
        linkedNode.key = emptyLinkedKeyNode;
        endNode(linkedNode, nextContext.offset, nextContext.startLoc);
        return {
          nextConsumeToken: token,
          node: linkedNode
        };
      }
    }
    endNode(linkedNode, tokenizer.currentOffset(), tokenizer.currentPosition());
    return {
      node: linkedNode
    };
  }
  function parseMessage(tokenizer) {
    const context = tokenizer.context();
    const startOffset = context.currentType === 1 ? tokenizer.currentOffset() : context.offset;
    const startLoc = context.currentType === 1 ? context.endLoc : context.startLoc;
    const node = startNode(2, startOffset, startLoc);
    node.items = [];
    let nextToken = null;
    let modulo = null;
    do {
      const token = nextToken || tokenizer.nextToken();
      nextToken = null;
      switch (token.type) {
        case 0:
          if (token.value == null) {
            emitError(tokenizer, CompileErrorCodes.UNEXPECTED_LEXICAL_ANALYSIS, context.lastStartLoc, 0, getTokenCaption(token));
          }
          node.items.push(parseText(tokenizer, token.value || ""));
          break;
        case 6:
          if (token.value == null) {
            emitError(tokenizer, CompileErrorCodes.UNEXPECTED_LEXICAL_ANALYSIS, context.lastStartLoc, 0, getTokenCaption(token));
          }
          node.items.push(parseList(tokenizer, token.value || ""));
          break;
        case 4:
          modulo = true;
          break;
        case 5:
          if (token.value == null) {
            emitError(tokenizer, CompileErrorCodes.UNEXPECTED_LEXICAL_ANALYSIS, context.lastStartLoc, 0, getTokenCaption(token));
          }
          node.items.push(parseNamed(tokenizer, token.value || "", !!modulo));
          if (modulo) {
            emitWarn(tokenizer, CompileWarnCodes.USE_MODULO_SYNTAX, context.lastStartLoc, 0, getTokenCaption(token));
            modulo = null;
          }
          break;
        case 7:
          if (token.value == null) {
            emitError(tokenizer, CompileErrorCodes.UNEXPECTED_LEXICAL_ANALYSIS, context.lastStartLoc, 0, getTokenCaption(token));
          }
          node.items.push(parseLiteral(tokenizer, token.value || ""));
          break;
        case 8: {
          const parsed = parseLinked(tokenizer);
          node.items.push(parsed.node);
          nextToken = parsed.nextConsumeToken || null;
          break;
        }
      }
    } while (context.currentType !== 14 && context.currentType !== 1);
    const endOffset = context.currentType === 1 ? context.lastOffset : tokenizer.currentOffset();
    const endLoc = context.currentType === 1 ? context.lastEndLoc : tokenizer.currentPosition();
    endNode(node, endOffset, endLoc);
    return node;
  }
  function parsePlural(tokenizer, offset, loc, msgNode) {
    const context = tokenizer.context();
    let hasEmptyMessage = msgNode.items.length === 0;
    const node = startNode(1, offset, loc);
    node.cases = [];
    node.cases.push(msgNode);
    do {
      const msg = parseMessage(tokenizer);
      if (!hasEmptyMessage) {
        hasEmptyMessage = msg.items.length === 0;
      }
      node.cases.push(msg);
    } while (context.currentType !== 14);
    if (hasEmptyMessage) {
      emitError(tokenizer, CompileErrorCodes.MUST_HAVE_MESSAGES_IN_PLURAL, loc, 0);
    }
    endNode(node, tokenizer.currentOffset(), tokenizer.currentPosition());
    return node;
  }
  function parseResource(tokenizer) {
    const context = tokenizer.context();
    const { offset, startLoc } = context;
    const msgNode = parseMessage(tokenizer);
    if (context.currentType === 14) {
      return msgNode;
    } else {
      return parsePlural(tokenizer, offset, startLoc, msgNode);
    }
  }
  function parse2(source) {
    const tokenizer = createTokenizer(source, assign({}, options));
    const context = tokenizer.context();
    const node = startNode(0, context.offset, context.startLoc);
    if (location2 && node.loc) {
      node.loc.source = source;
    }
    node.body = parseResource(tokenizer);
    if (options.onCacheKey) {
      node.cacheKey = options.onCacheKey(source);
    }
    if (context.currentType !== 14) {
      emitError(tokenizer, CompileErrorCodes.UNEXPECTED_LEXICAL_ANALYSIS, context.lastStartLoc, 0, source[context.offset] || "");
    }
    endNode(node, tokenizer.currentOffset(), tokenizer.currentPosition());
    return node;
  }
  return { parse: parse2 };
}
function getTokenCaption(token) {
  if (token.type === 14) {
    return "EOF";
  }
  const name = (token.value || "").replace(/\r?\n/gu, "\\n");
  return name.length > 10 ? name.slice(0, 9) + "…" : name;
}
function createTransformer(ast, options = {}) {
  const _context = {
    ast,
    helpers: /* @__PURE__ */ new Set()
  };
  const context = () => _context;
  const helper = (name) => {
    _context.helpers.add(name);
    return name;
  };
  return { context, helper };
}
function traverseNodes(nodes, transformer) {
  for (let i = 0; i < nodes.length; i++) {
    traverseNode(nodes[i], transformer);
  }
}
function traverseNode(node, transformer) {
  switch (node.type) {
    case 1:
      traverseNodes(node.cases, transformer);
      transformer.helper(
        "plural"
        /* HelperNameMap.PLURAL */
      );
      break;
    case 2:
      traverseNodes(node.items, transformer);
      break;
    case 6: {
      const linked = node;
      traverseNode(linked.key, transformer);
      transformer.helper(
        "linked"
        /* HelperNameMap.LINKED */
      );
      transformer.helper(
        "type"
        /* HelperNameMap.TYPE */
      );
      break;
    }
    case 5:
      transformer.helper(
        "interpolate"
        /* HelperNameMap.INTERPOLATE */
      );
      transformer.helper(
        "list"
        /* HelperNameMap.LIST */
      );
      break;
    case 4:
      transformer.helper(
        "interpolate"
        /* HelperNameMap.INTERPOLATE */
      );
      transformer.helper(
        "named"
        /* HelperNameMap.NAMED */
      );
      break;
  }
}
function transform(ast, options = {}) {
  const transformer = createTransformer(ast);
  transformer.helper(
    "normalize"
    /* HelperNameMap.NORMALIZE */
  );
  ast.body && traverseNode(ast.body, transformer);
  const context = transformer.context();
  ast.helpers = Array.from(context.helpers);
}
function optimize(ast) {
  const body = ast.body;
  if (body.type === 2) {
    optimizeMessageNode(body);
  } else {
    body.cases.forEach((c) => optimizeMessageNode(c));
  }
  return ast;
}
function optimizeMessageNode(message) {
  if (message.items.length === 1) {
    const item = message.items[0];
    if (item.type === 3 || item.type === 9) {
      message.static = item.value;
      delete item.value;
    }
  } else {
    const values = [];
    for (let i = 0; i < message.items.length; i++) {
      const item = message.items[i];
      if (!(item.type === 3 || item.type === 9)) {
        break;
      }
      if (item.value == null) {
        break;
      }
      values.push(item.value);
    }
    if (values.length === message.items.length) {
      message.static = join(values);
      for (let i = 0; i < message.items.length; i++) {
        const item = message.items[i];
        if (item.type === 3 || item.type === 9) {
          delete item.value;
        }
      }
    }
  }
}
const ERROR_DOMAIN$1 = "minifier";
function minify(node) {
  node.t = node.type;
  switch (node.type) {
    case 0: {
      const resource = node;
      minify(resource.body);
      resource.b = resource.body;
      delete resource.body;
      break;
    }
    case 1: {
      const plural = node;
      const cases = plural.cases;
      for (let i = 0; i < cases.length; i++) {
        minify(cases[i]);
      }
      plural.c = cases;
      delete plural.cases;
      break;
    }
    case 2: {
      const message = node;
      const items = message.items;
      for (let i = 0; i < items.length; i++) {
        minify(items[i]);
      }
      message.i = items;
      delete message.items;
      if (message.static) {
        message.s = message.static;
        delete message.static;
      }
      break;
    }
    case 3:
    case 9:
    case 8:
    case 7: {
      const valueNode = node;
      if (valueNode.value) {
        valueNode.v = valueNode.value;
        delete valueNode.value;
      }
      break;
    }
    case 6: {
      const linked = node;
      minify(linked.key);
      linked.k = linked.key;
      delete linked.key;
      if (linked.modifier) {
        minify(linked.modifier);
        linked.m = linked.modifier;
        delete linked.modifier;
      }
      break;
    }
    case 5: {
      const list = node;
      list.i = list.index;
      delete list.index;
      break;
    }
    case 4: {
      const named = node;
      named.k = named.key;
      delete named.key;
      break;
    }
    default: {
      throw createCompileError(CompileErrorCodes.UNHANDLED_MINIFIER_NODE_TYPE, null, {
        domain: ERROR_DOMAIN$1,
        args: [node.type]
      });
    }
  }
  delete node.type;
}
const ERROR_DOMAIN = "parser";
function createCodeGenerator(ast, options) {
  const { filename, breakLineCode, needIndent: _needIndent } = options;
  const location2 = options.location !== false;
  const _context = {
    filename,
    code: "",
    column: 1,
    line: 1,
    offset: 0,
    map: void 0,
    breakLineCode,
    needIndent: _needIndent,
    indentLevel: 0
  };
  if (location2 && ast.loc) {
    _context.source = ast.loc.source;
  }
  const context = () => _context;
  function push(code2, node) {
    _context.code += code2;
  }
  function _newline(n, withBreakLine = true) {
    const _breakLineCode = withBreakLine ? breakLineCode : "";
    push(_needIndent ? _breakLineCode + `  `.repeat(n) : _breakLineCode);
  }
  function indent(withNewLine = true) {
    const level = ++_context.indentLevel;
    withNewLine && _newline(level);
  }
  function deindent(withNewLine = true) {
    const level = --_context.indentLevel;
    withNewLine && _newline(level);
  }
  function newline() {
    _newline(_context.indentLevel);
  }
  const helper = (key) => `_${key}`;
  const needIndent = () => _context.needIndent;
  return {
    context,
    push,
    indent,
    deindent,
    newline,
    helper,
    needIndent
  };
}
function generateLinkedNode(generator, node) {
  const { helper } = generator;
  generator.push(`${helper(
    "linked"
    /* HelperNameMap.LINKED */
  )}(`);
  generateNode(generator, node.key);
  if (node.modifier) {
    generator.push(`, `);
    generateNode(generator, node.modifier);
    generator.push(`, _type`);
  } else {
    generator.push(`, undefined, _type`);
  }
  generator.push(`)`);
}
function generateMessageNode(generator, node) {
  const { helper, needIndent } = generator;
  generator.push(`${helper(
    "normalize"
    /* HelperNameMap.NORMALIZE */
  )}([`);
  generator.indent(needIndent());
  const length = node.items.length;
  for (let i = 0; i < length; i++) {
    generateNode(generator, node.items[i]);
    if (i === length - 1) {
      break;
    }
    generator.push(", ");
  }
  generator.deindent(needIndent());
  generator.push("])");
}
function generatePluralNode(generator, node) {
  const { helper, needIndent } = generator;
  if (node.cases.length > 1) {
    generator.push(`${helper(
      "plural"
      /* HelperNameMap.PLURAL */
    )}([`);
    generator.indent(needIndent());
    const length = node.cases.length;
    for (let i = 0; i < length; i++) {
      generateNode(generator, node.cases[i]);
      if (i === length - 1) {
        break;
      }
      generator.push(", ");
    }
    generator.deindent(needIndent());
    generator.push(`])`);
  }
}
function generateResource(generator, node) {
  if (node.body) {
    generateNode(generator, node.body);
  } else {
    generator.push("null");
  }
}
function generateNode(generator, node) {
  const { helper } = generator;
  switch (node.type) {
    case 0:
      generateResource(generator, node);
      break;
    case 1:
      generatePluralNode(generator, node);
      break;
    case 2:
      generateMessageNode(generator, node);
      break;
    case 6:
      generateLinkedNode(generator, node);
      break;
    case 8:
      generator.push(JSON.stringify(node.value), node);
      break;
    case 7:
      generator.push(JSON.stringify(node.value), node);
      break;
    case 5:
      generator.push(`${helper(
        "interpolate"
        /* HelperNameMap.INTERPOLATE */
      )}(${helper(
        "list"
        /* HelperNameMap.LIST */
      )}(${node.index}))`, node);
      break;
    case 4:
      generator.push(`${helper(
        "interpolate"
        /* HelperNameMap.INTERPOLATE */
      )}(${helper(
        "named"
        /* HelperNameMap.NAMED */
      )}(${JSON.stringify(node.key)}))`, node);
      break;
    case 9:
      generator.push(JSON.stringify(node.value), node);
      break;
    case 3:
      generator.push(JSON.stringify(node.value), node);
      break;
    default: {
      throw createCompileError(CompileErrorCodes.UNHANDLED_CODEGEN_NODE_TYPE, null, {
        domain: ERROR_DOMAIN,
        args: [node.type]
      });
    }
  }
}
const generate = (ast, options = {}) => {
  const mode = isString(options.mode) ? options.mode : "normal";
  const filename = isString(options.filename) ? options.filename : "message.intl";
  !!options.sourceMap;
  const breakLineCode = options.breakLineCode != null ? options.breakLineCode : mode === "arrow" ? ";" : "\n";
  const needIndent = options.needIndent ? options.needIndent : mode !== "arrow";
  const helpers = ast.helpers || [];
  const generator = createCodeGenerator(ast, {
    filename,
    breakLineCode,
    needIndent
  });
  generator.push(mode === "normal" ? `function __msg__ (ctx) {` : `(ctx) => {`);
  generator.indent(needIndent);
  if (helpers.length > 0) {
    generator.push(`const { ${join(helpers.map((s) => `${s}: _${s}`), ", ")} } = ctx`);
    generator.newline();
  }
  generator.push(`return `);
  generateNode(generator, ast);
  generator.deindent(needIndent);
  generator.push(`}`);
  delete ast.helpers;
  const { code: code2, map } = generator.context();
  return {
    ast,
    code: code2,
    map: map ? map.toJSON() : void 0
    // eslint-disable-line @typescript-eslint/no-explicit-any
  };
};
function baseCompile$1(source, options = {}) {
  const assignedOptions = assign({}, options);
  const jit = !!assignedOptions.jit;
  const enalbeMinify = !!assignedOptions.minify;
  const enambeOptimize = assignedOptions.optimize == null ? true : assignedOptions.optimize;
  const parser = createParser(assignedOptions);
  const ast = parser.parse(source);
  if (!jit) {
    transform(ast, assignedOptions);
    return generate(ast, assignedOptions);
  } else {
    enambeOptimize && optimize(ast);
    enalbeMinify && minify(ast);
    return { ast, code: "" };
  }
}
/*!
  * core-base v9.14.5
  * (c) 2025 kazuya kawaguchi
  * Released under the MIT License.
  */
function initFeatureFlags$1() {
  if (typeof __INTLIFY_PROD_DEVTOOLS__ !== "boolean") {
    getGlobalThis().__INTLIFY_PROD_DEVTOOLS__ = false;
  }
  if (typeof __INTLIFY_JIT_COMPILATION__ !== "boolean") {
    getGlobalThis().__INTLIFY_JIT_COMPILATION__ = false;
  }
  if (typeof __INTLIFY_DROP_MESSAGE_COMPILER__ !== "boolean") {
    getGlobalThis().__INTLIFY_DROP_MESSAGE_COMPILER__ = false;
  }
}
function isMessageAST(val) {
  return isObject$1(val) && resolveType(val) === 0 && (hasOwn(val, "b") || hasOwn(val, "body"));
}
const PROPS_BODY = ["b", "body"];
function resolveBody(node) {
  return resolveProps(node, PROPS_BODY);
}
const PROPS_CASES = ["c", "cases"];
function resolveCases(node) {
  return resolveProps(node, PROPS_CASES, []);
}
const PROPS_STATIC = ["s", "static"];
function resolveStatic(node) {
  return resolveProps(node, PROPS_STATIC);
}
const PROPS_ITEMS = ["i", "items"];
function resolveItems(node) {
  return resolveProps(node, PROPS_ITEMS, []);
}
const PROPS_TYPE = ["t", "type"];
function resolveType(node) {
  return resolveProps(node, PROPS_TYPE);
}
const PROPS_VALUE = ["v", "value"];
function resolveValue$1(node, type) {
  const resolved = resolveProps(node, PROPS_VALUE);
  if (resolved != null) {
    return resolved;
  } else {
    throw createUnhandleNodeError(type);
  }
}
const PROPS_MODIFIER = ["m", "modifier"];
function resolveLinkedModifier(node) {
  return resolveProps(node, PROPS_MODIFIER);
}
const PROPS_KEY = ["k", "key"];
function resolveLinkedKey(node) {
  const resolved = resolveProps(node, PROPS_KEY);
  if (resolved) {
    return resolved;
  } else {
    throw createUnhandleNodeError(
      6
      /* NodeTypes.Linked */
    );
  }
}
function resolveProps(node, props, defaultValue) {
  for (let i = 0; i < props.length; i++) {
    const prop = props[i];
    if (hasOwn(node, prop) && node[prop] != null) {
      return node[prop];
    }
  }
  return defaultValue;
}
const AST_NODE_PROPS_KEYS = [
  ...PROPS_BODY,
  ...PROPS_CASES,
  ...PROPS_STATIC,
  ...PROPS_ITEMS,
  ...PROPS_KEY,
  ...PROPS_MODIFIER,
  ...PROPS_VALUE,
  ...PROPS_TYPE
];
function createUnhandleNodeError(type) {
  return new Error(`unhandled node type: ${type}`);
}
const pathStateMachine = [];
pathStateMachine[
  0
  /* States.BEFORE_PATH */
] = {
  [
    "w"
    /* PathCharTypes.WORKSPACE */
  ]: [
    0
    /* States.BEFORE_PATH */
  ],
  [
    "i"
    /* PathCharTypes.IDENT */
  ]: [
    3,
    0
    /* Actions.APPEND */
  ],
  [
    "["
    /* PathCharTypes.LEFT_BRACKET */
  ]: [
    4
    /* States.IN_SUB_PATH */
  ],
  [
    "o"
    /* PathCharTypes.END_OF_FAIL */
  ]: [
    7
    /* States.AFTER_PATH */
  ]
};
pathStateMachine[
  1
  /* States.IN_PATH */
] = {
  [
    "w"
    /* PathCharTypes.WORKSPACE */
  ]: [
    1
    /* States.IN_PATH */
  ],
  [
    "."
    /* PathCharTypes.DOT */
  ]: [
    2
    /* States.BEFORE_IDENT */
  ],
  [
    "["
    /* PathCharTypes.LEFT_BRACKET */
  ]: [
    4
    /* States.IN_SUB_PATH */
  ],
  [
    "o"
    /* PathCharTypes.END_OF_FAIL */
  ]: [
    7
    /* States.AFTER_PATH */
  ]
};
pathStateMachine[
  2
  /* States.BEFORE_IDENT */
] = {
  [
    "w"
    /* PathCharTypes.WORKSPACE */
  ]: [
    2
    /* States.BEFORE_IDENT */
  ],
  [
    "i"
    /* PathCharTypes.IDENT */
  ]: [
    3,
    0
    /* Actions.APPEND */
  ],
  [
    "0"
    /* PathCharTypes.ZERO */
  ]: [
    3,
    0
    /* Actions.APPEND */
  ]
};
pathStateMachine[
  3
  /* States.IN_IDENT */
] = {
  [
    "i"
    /* PathCharTypes.IDENT */
  ]: [
    3,
    0
    /* Actions.APPEND */
  ],
  [
    "0"
    /* PathCharTypes.ZERO */
  ]: [
    3,
    0
    /* Actions.APPEND */
  ],
  [
    "w"
    /* PathCharTypes.WORKSPACE */
  ]: [
    1,
    1
    /* Actions.PUSH */
  ],
  [
    "."
    /* PathCharTypes.DOT */
  ]: [
    2,
    1
    /* Actions.PUSH */
  ],
  [
    "["
    /* PathCharTypes.LEFT_BRACKET */
  ]: [
    4,
    1
    /* Actions.PUSH */
  ],
  [
    "o"
    /* PathCharTypes.END_OF_FAIL */
  ]: [
    7,
    1
    /* Actions.PUSH */
  ]
};
pathStateMachine[
  4
  /* States.IN_SUB_PATH */
] = {
  [
    "'"
    /* PathCharTypes.SINGLE_QUOTE */
  ]: [
    5,
    0
    /* Actions.APPEND */
  ],
  [
    '"'
    /* PathCharTypes.DOUBLE_QUOTE */
  ]: [
    6,
    0
    /* Actions.APPEND */
  ],
  [
    "["
    /* PathCharTypes.LEFT_BRACKET */
  ]: [
    4,
    2
    /* Actions.INC_SUB_PATH_DEPTH */
  ],
  [
    "]"
    /* PathCharTypes.RIGHT_BRACKET */
  ]: [
    1,
    3
    /* Actions.PUSH_SUB_PATH */
  ],
  [
    "o"
    /* PathCharTypes.END_OF_FAIL */
  ]: 8,
  [
    "l"
    /* PathCharTypes.ELSE */
  ]: [
    4,
    0
    /* Actions.APPEND */
  ]
};
pathStateMachine[
  5
  /* States.IN_SINGLE_QUOTE */
] = {
  [
    "'"
    /* PathCharTypes.SINGLE_QUOTE */
  ]: [
    4,
    0
    /* Actions.APPEND */
  ],
  [
    "o"
    /* PathCharTypes.END_OF_FAIL */
  ]: 8,
  [
    "l"
    /* PathCharTypes.ELSE */
  ]: [
    5,
    0
    /* Actions.APPEND */
  ]
};
pathStateMachine[
  6
  /* States.IN_DOUBLE_QUOTE */
] = {
  [
    '"'
    /* PathCharTypes.DOUBLE_QUOTE */
  ]: [
    4,
    0
    /* Actions.APPEND */
  ],
  [
    "o"
    /* PathCharTypes.END_OF_FAIL */
  ]: 8,
  [
    "l"
    /* PathCharTypes.ELSE */
  ]: [
    6,
    0
    /* Actions.APPEND */
  ]
};
const literalValueRE = /^\s?(?:true|false|-?[\d.]+|'[^']*'|"[^"]*")\s?$/;
function isLiteral(exp) {
  return literalValueRE.test(exp);
}
function stripQuotes(str) {
  const a = str.charCodeAt(0);
  const b = str.charCodeAt(str.length - 1);
  return a === b && (a === 34 || a === 39) ? str.slice(1, -1) : str;
}
function getPathCharType(ch) {
  if (ch === void 0 || ch === null) {
    return "o";
  }
  const code2 = ch.charCodeAt(0);
  switch (code2) {
    case 91:
    case 93:
    case 46:
    case 34:
    case 39:
      return ch;
    case 95:
    case 36:
    case 45:
      return "i";
    case 9:
    case 10:
    case 13:
    case 160:
    case 65279:
    case 8232:
    case 8233:
      return "w";
  }
  return "i";
}
function formatSubPath(path) {
  const trimmed = path.trim();
  if (path.charAt(0) === "0" && isNaN(parseInt(path))) {
    return false;
  }
  return isLiteral(trimmed) ? stripQuotes(trimmed) : "*" + trimmed;
}
function parse(path) {
  const keys = [];
  let index = -1;
  let mode = 0;
  let subPathDepth = 0;
  let c;
  let key;
  let newChar;
  let type;
  let transition;
  let action;
  let typeMap;
  const actions = [];
  actions[
    0
    /* Actions.APPEND */
  ] = () => {
    if (key === void 0) {
      key = newChar;
    } else {
      key += newChar;
    }
  };
  actions[
    1
    /* Actions.PUSH */
  ] = () => {
    if (key !== void 0) {
      keys.push(key);
      key = void 0;
    }
  };
  actions[
    2
    /* Actions.INC_SUB_PATH_DEPTH */
  ] = () => {
    actions[
      0
      /* Actions.APPEND */
    ]();
    subPathDepth++;
  };
  actions[
    3
    /* Actions.PUSH_SUB_PATH */
  ] = () => {
    if (subPathDepth > 0) {
      subPathDepth--;
      mode = 4;
      actions[
        0
        /* Actions.APPEND */
      ]();
    } else {
      subPathDepth = 0;
      if (key === void 0) {
        return false;
      }
      key = formatSubPath(key);
      if (key === false) {
        return false;
      } else {
        actions[
          1
          /* Actions.PUSH */
        ]();
      }
    }
  };
  function maybeUnescapeQuote() {
    const nextChar = path[index + 1];
    if (mode === 5 && nextChar === "'" || mode === 6 && nextChar === '"') {
      index++;
      newChar = "\\" + nextChar;
      actions[
        0
        /* Actions.APPEND */
      ]();
      return true;
    }
  }
  while (mode !== null) {
    index++;
    c = path[index];
    if (c === "\\" && maybeUnescapeQuote()) {
      continue;
    }
    type = getPathCharType(c);
    typeMap = pathStateMachine[mode];
    transition = typeMap[type] || typeMap[
      "l"
      /* PathCharTypes.ELSE */
    ] || 8;
    if (transition === 8) {
      return;
    }
    mode = transition[0];
    if (transition[1] !== void 0) {
      action = actions[transition[1]];
      if (action) {
        newChar = c;
        if (action() === false) {
          return;
        }
      }
    }
    if (mode === 7) {
      return keys;
    }
  }
}
const cache = /* @__PURE__ */ new Map();
function resolveWithKeyValue(obj, path) {
  return isObject$1(obj) ? obj[path] : null;
}
function resolveValue(obj, path) {
  if (!isObject$1(obj)) {
    return null;
  }
  let hit = cache.get(path);
  if (!hit) {
    hit = parse(path);
    if (hit) {
      cache.set(path, hit);
    }
  }
  if (!hit) {
    return null;
  }
  const len = hit.length;
  let last = obj;
  let i = 0;
  while (i < len) {
    const key = hit[i];
    if (AST_NODE_PROPS_KEYS.includes(key) && isMessageAST(last)) {
      return null;
    }
    const val = last[key];
    if (val === void 0) {
      return null;
    }
    if (isFunction(last)) {
      return null;
    }
    last = val;
    i++;
  }
  return last;
}
const DEFAULT_MODIFIER = (str) => str;
const DEFAULT_MESSAGE = (ctx) => "";
const DEFAULT_MESSAGE_DATA_TYPE = "text";
const DEFAULT_NORMALIZE = (values) => values.length === 0 ? "" : join$1(values);
const DEFAULT_INTERPOLATE = toDisplayString;
function pluralDefault(choice, choicesLength) {
  choice = Math.abs(choice);
  if (choicesLength === 2) {
    return choice ? choice > 1 ? 1 : 0 : 1;
  }
  return choice ? Math.min(choice, 2) : 0;
}
function getPluralIndex(options) {
  const index = isNumber(options.pluralIndex) ? options.pluralIndex : -1;
  return options.named && (isNumber(options.named.count) || isNumber(options.named.n)) ? isNumber(options.named.count) ? options.named.count : isNumber(options.named.n) ? options.named.n : index : index;
}
function normalizeNamed(pluralIndex, props) {
  if (!props.count) {
    props.count = pluralIndex;
  }
  if (!props.n) {
    props.n = pluralIndex;
  }
}
function createMessageContext(options = {}) {
  const locale = options.locale;
  const pluralIndex = getPluralIndex(options);
  const pluralRule = isObject$1(options.pluralRules) && isString$1(locale) && isFunction(options.pluralRules[locale]) ? options.pluralRules[locale] : pluralDefault;
  const orgPluralRule = isObject$1(options.pluralRules) && isString$1(locale) && isFunction(options.pluralRules[locale]) ? pluralDefault : void 0;
  const plural = (messages) => {
    return messages[pluralRule(pluralIndex, messages.length, orgPluralRule)];
  };
  const _list = options.list || [];
  const list = (index) => _list[index];
  const _named = options.named || create();
  isNumber(options.pluralIndex) && normalizeNamed(pluralIndex, _named);
  const named = (key) => _named[key];
  function message(key) {
    const msg = isFunction(options.messages) ? options.messages(key) : isObject$1(options.messages) ? options.messages[key] : false;
    return !msg ? options.parent ? options.parent.message(key) : DEFAULT_MESSAGE : msg;
  }
  const _modifier = (name) => options.modifiers ? options.modifiers[name] : DEFAULT_MODIFIER;
  const normalize = isPlainObject(options.processor) && isFunction(options.processor.normalize) ? options.processor.normalize : DEFAULT_NORMALIZE;
  const interpolate = isPlainObject(options.processor) && isFunction(options.processor.interpolate) ? options.processor.interpolate : DEFAULT_INTERPOLATE;
  const type = isPlainObject(options.processor) && isString$1(options.processor.type) ? options.processor.type : DEFAULT_MESSAGE_DATA_TYPE;
  const linked = (key, ...args) => {
    const [arg1, arg2] = args;
    let type2 = "text";
    let modifier = "";
    if (args.length === 1) {
      if (isObject$1(arg1)) {
        modifier = arg1.modifier || modifier;
        type2 = arg1.type || type2;
      } else if (isString$1(arg1)) {
        modifier = arg1 || modifier;
      }
    } else if (args.length === 2) {
      if (isString$1(arg1)) {
        modifier = arg1 || modifier;
      }
      if (isString$1(arg2)) {
        type2 = arg2 || type2;
      }
    }
    const ret = message(key)(ctx);
    const msg = (
      // The message in vnode resolved with linked are returned as an array by processor.nomalize
      type2 === "vnode" && isArray(ret) && modifier ? ret[0] : ret
    );
    return modifier ? _modifier(modifier)(msg, type2) : msg;
  };
  const ctx = {
    [
      "list"
      /* HelperNameMap.LIST */
    ]: list,
    [
      "named"
      /* HelperNameMap.NAMED */
    ]: named,
    [
      "plural"
      /* HelperNameMap.PLURAL */
    ]: plural,
    [
      "linked"
      /* HelperNameMap.LINKED */
    ]: linked,
    [
      "message"
      /* HelperNameMap.MESSAGE */
    ]: message,
    [
      "type"
      /* HelperNameMap.TYPE */
    ]: type,
    [
      "interpolate"
      /* HelperNameMap.INTERPOLATE */
    ]: interpolate,
    [
      "normalize"
      /* HelperNameMap.NORMALIZE */
    ]: normalize,
    [
      "values"
      /* HelperNameMap.VALUES */
    ]: assign$1(create(), _list, _named)
  };
  return ctx;
}
let devtools = null;
function setDevToolsHook(hook) {
  devtools = hook;
}
function initI18nDevTools(i18n2, version2, meta) {
  devtools && devtools.emit("i18n:init", {
    timestamp: Date.now(),
    i18n: i18n2,
    version: version2,
    meta
  });
}
const translateDevTools = /* @__PURE__ */ createDevToolsHook(
  "function:translate"
  /* IntlifyDevToolsHooks.FunctionTranslate */
);
function createDevToolsHook(hook) {
  return (payloads) => devtools && devtools.emit(hook, payloads);
}
const code$1$1 = CompileWarnCodes.__EXTEND_POINT__;
const inc$1$1 = incrementer(code$1$1);
const CoreWarnCodes = {
  // 2
  FALLBACK_TO_TRANSLATE: inc$1$1(),
  // 3
  CANNOT_FORMAT_NUMBER: inc$1$1(),
  // 4
  FALLBACK_TO_NUMBER_FORMAT: inc$1$1(),
  // 5
  CANNOT_FORMAT_DATE: inc$1$1(),
  // 6
  FALLBACK_TO_DATE_FORMAT: inc$1$1(),
  // 7
  EXPERIMENTAL_CUSTOM_MESSAGE_COMPILER: inc$1$1(),
  // 8
  __EXTEND_POINT__: inc$1$1()
  // 9
};
const code$2 = CompileErrorCodes.__EXTEND_POINT__;
const inc$2 = incrementer(code$2);
const CoreErrorCodes = {
  INVALID_ARGUMENT: code$2,
  // 17
  INVALID_DATE_ARGUMENT: inc$2(),
  // 18
  INVALID_ISO_DATE_ARGUMENT: inc$2(),
  // 19
  NOT_SUPPORT_NON_STRING_MESSAGE: inc$2(),
  // 20
  NOT_SUPPORT_LOCALE_PROMISE_VALUE: inc$2(),
  // 21
  NOT_SUPPORT_LOCALE_ASYNC_FUNCTION: inc$2(),
  // 22
  NOT_SUPPORT_LOCALE_TYPE: inc$2(),
  // 23
  __EXTEND_POINT__: inc$2()
  // 24
};
function createCoreError(code2) {
  return createCompileError(code2, null, void 0);
}
function getLocale(context, options) {
  return options.locale != null ? resolveLocale(options.locale) : resolveLocale(context.locale);
}
let _resolveLocale;
function resolveLocale(locale) {
  if (isString$1(locale)) {
    return locale;
  } else {
    if (isFunction(locale)) {
      if (locale.resolvedOnce && _resolveLocale != null) {
        return _resolveLocale;
      } else if (locale.constructor.name === "Function") {
        const resolve2 = locale();
        if (isPromise(resolve2)) {
          throw createCoreError(CoreErrorCodes.NOT_SUPPORT_LOCALE_PROMISE_VALUE);
        }
        return _resolveLocale = resolve2;
      } else {
        throw createCoreError(CoreErrorCodes.NOT_SUPPORT_LOCALE_ASYNC_FUNCTION);
      }
    } else {
      throw createCoreError(CoreErrorCodes.NOT_SUPPORT_LOCALE_TYPE);
    }
  }
}
function fallbackWithSimple(ctx, fallback, start) {
  return [.../* @__PURE__ */ new Set([
    start,
    ...isArray(fallback) ? fallback : isObject$1(fallback) ? Object.keys(fallback) : isString$1(fallback) ? [fallback] : [start]
  ])];
}
function fallbackWithLocaleChain(ctx, fallback, start) {
  const startLocale = isString$1(start) ? start : DEFAULT_LOCALE;
  const context = ctx;
  if (!context.__localeChainCache) {
    context.__localeChainCache = /* @__PURE__ */ new Map();
  }
  let chain = context.__localeChainCache.get(startLocale);
  if (!chain) {
    chain = [];
    let block = [start];
    while (isArray(block)) {
      block = appendBlockToChain(chain, block, fallback);
    }
    const defaults = isArray(fallback) || !isPlainObject(fallback) ? fallback : fallback["default"] ? fallback["default"] : null;
    block = isString$1(defaults) ? [defaults] : defaults;
    if (isArray(block)) {
      appendBlockToChain(chain, block, false);
    }
    context.__localeChainCache.set(startLocale, chain);
  }
  return chain;
}
function appendBlockToChain(chain, block, blocks) {
  let follow = true;
  for (let i = 0; i < block.length && isBoolean(follow); i++) {
    const locale = block[i];
    if (isString$1(locale)) {
      follow = appendLocaleToChain(chain, block[i], blocks);
    }
  }
  return follow;
}
function appendLocaleToChain(chain, locale, blocks) {
  let follow;
  const tokens = locale.split("-");
  do {
    const target = tokens.join("-");
    follow = appendItemToChain(chain, target, blocks);
    tokens.splice(-1, 1);
  } while (tokens.length && follow === true);
  return follow;
}
function appendItemToChain(chain, target, blocks) {
  let follow = false;
  if (!chain.includes(target)) {
    follow = true;
    if (target) {
      follow = target[target.length - 1] !== "!";
      const locale = target.replace(/!/g, "");
      chain.push(locale);
      if ((isArray(blocks) || isPlainObject(blocks)) && blocks[locale]) {
        follow = blocks[locale];
      }
    }
  }
  return follow;
}
const VERSION$1 = "9.14.5";
const NOT_REOSLVED = -1;
const DEFAULT_LOCALE = "en-US";
const MISSING_RESOLVE_VALUE = "";
const capitalize = (str) => `${str.charAt(0).toLocaleUpperCase()}${str.substr(1)}`;
function getDefaultLinkedModifiers() {
  return {
    upper: (val, type) => {
      return type === "text" && isString$1(val) ? val.toUpperCase() : type === "vnode" && isObject$1(val) && "__v_isVNode" in val ? val.children.toUpperCase() : val;
    },
    lower: (val, type) => {
      return type === "text" && isString$1(val) ? val.toLowerCase() : type === "vnode" && isObject$1(val) && "__v_isVNode" in val ? val.children.toLowerCase() : val;
    },
    capitalize: (val, type) => {
      return type === "text" && isString$1(val) ? capitalize(val) : type === "vnode" && isObject$1(val) && "__v_isVNode" in val ? capitalize(val.children) : val;
    }
  };
}
let _compiler;
function registerMessageCompiler(compiler) {
  _compiler = compiler;
}
let _resolver;
function registerMessageResolver(resolver) {
  _resolver = resolver;
}
let _fallbacker;
function registerLocaleFallbacker(fallbacker) {
  _fallbacker = fallbacker;
}
let _additionalMeta = null;
const setAdditionalMeta = /* @__NO_SIDE_EFFECTS__ */ (meta) => {
  _additionalMeta = meta;
};
const getAdditionalMeta = /* @__NO_SIDE_EFFECTS__ */ () => _additionalMeta;
let _fallbackContext = null;
const setFallbackContext = (context) => {
  _fallbackContext = context;
};
const getFallbackContext = () => _fallbackContext;
let _cid = 0;
function createCoreContext(options = {}) {
  const onWarn = isFunction(options.onWarn) ? options.onWarn : warn;
  const version2 = isString$1(options.version) ? options.version : VERSION$1;
  const locale = isString$1(options.locale) || isFunction(options.locale) ? options.locale : DEFAULT_LOCALE;
  const _locale = isFunction(locale) ? DEFAULT_LOCALE : locale;
  const fallbackLocale = isArray(options.fallbackLocale) || isPlainObject(options.fallbackLocale) || isString$1(options.fallbackLocale) || options.fallbackLocale === false ? options.fallbackLocale : _locale;
  const messages = isPlainObject(options.messages) ? options.messages : createResources(_locale);
  const datetimeFormats = isPlainObject(options.datetimeFormats) ? options.datetimeFormats : createResources(_locale);
  const numberFormats = isPlainObject(options.numberFormats) ? options.numberFormats : createResources(_locale);
  const modifiers = assign$1(create(), options.modifiers, getDefaultLinkedModifiers());
  const pluralRules = options.pluralRules || create();
  const missing = isFunction(options.missing) ? options.missing : null;
  const missingWarn = isBoolean(options.missingWarn) || isRegExp(options.missingWarn) ? options.missingWarn : true;
  const fallbackWarn = isBoolean(options.fallbackWarn) || isRegExp(options.fallbackWarn) ? options.fallbackWarn : true;
  const fallbackFormat = !!options.fallbackFormat;
  const unresolving = !!options.unresolving;
  const postTranslation = isFunction(options.postTranslation) ? options.postTranslation : null;
  const processor = isPlainObject(options.processor) ? options.processor : null;
  const warnHtmlMessage = isBoolean(options.warnHtmlMessage) ? options.warnHtmlMessage : true;
  const escapeParameter = !!options.escapeParameter;
  const messageCompiler = isFunction(options.messageCompiler) ? options.messageCompiler : _compiler;
  const messageResolver = isFunction(options.messageResolver) ? options.messageResolver : _resolver || resolveWithKeyValue;
  const localeFallbacker = isFunction(options.localeFallbacker) ? options.localeFallbacker : _fallbacker || fallbackWithSimple;
  const fallbackContext = isObject$1(options.fallbackContext) ? options.fallbackContext : void 0;
  const internalOptions = options;
  const __datetimeFormatters = isObject$1(internalOptions.__datetimeFormatters) ? internalOptions.__datetimeFormatters : /* @__PURE__ */ new Map();
  const __numberFormatters = isObject$1(internalOptions.__numberFormatters) ? internalOptions.__numberFormatters : /* @__PURE__ */ new Map();
  const __meta = isObject$1(internalOptions.__meta) ? internalOptions.__meta : {};
  _cid++;
  const context = {
    version: version2,
    cid: _cid,
    locale,
    fallbackLocale,
    messages,
    modifiers,
    pluralRules,
    missing,
    missingWarn,
    fallbackWarn,
    fallbackFormat,
    unresolving,
    postTranslation,
    processor,
    warnHtmlMessage,
    escapeParameter,
    messageCompiler,
    messageResolver,
    localeFallbacker,
    fallbackContext,
    onWarn,
    __meta
  };
  {
    context.datetimeFormats = datetimeFormats;
    context.numberFormats = numberFormats;
    context.__datetimeFormatters = __datetimeFormatters;
    context.__numberFormatters = __numberFormatters;
  }
  if (__INTLIFY_PROD_DEVTOOLS__) {
    initI18nDevTools(context, version2, __meta);
  }
  return context;
}
const createResources = (locale) => ({ [locale]: create() });
function handleMissing(context, key, locale, missingWarn, type) {
  const { missing, onWarn } = context;
  if (missing !== null) {
    const ret = missing(context, locale, key, type);
    return isString$1(ret) ? ret : key;
  } else {
    return key;
  }
}
function updateFallbackLocale(ctx, locale, fallback) {
  const context = ctx;
  context.__localeChainCache = /* @__PURE__ */ new Map();
  ctx.localeFallbacker(ctx, fallback, locale);
}
function isAlmostSameLocale(locale, compareLocale) {
  if (locale === compareLocale)
    return false;
  return locale.split("-")[0] === compareLocale.split("-")[0];
}
function isImplicitFallback(targetLocale, locales) {
  const index = locales.indexOf(targetLocale);
  if (index === -1) {
    return false;
  }
  for (let i = index + 1; i < locales.length; i++) {
    if (isAlmostSameLocale(targetLocale, locales[i])) {
      return true;
    }
  }
  return false;
}
function format(ast) {
  const msg = (ctx) => formatParts(ctx, ast);
  return msg;
}
function formatParts(ctx, ast) {
  const body = resolveBody(ast);
  if (body == null) {
    throw createUnhandleNodeError(
      0
      /* NodeTypes.Resource */
    );
  }
  const type = resolveType(body);
  if (type === 1) {
    const plural = body;
    const cases = resolveCases(plural);
    return ctx.plural(cases.reduce((messages, c) => [
      ...messages,
      formatMessageParts(ctx, c)
    ], []));
  } else {
    return formatMessageParts(ctx, body);
  }
}
function formatMessageParts(ctx, node) {
  const static_ = resolveStatic(node);
  if (static_ != null) {
    return ctx.type === "text" ? static_ : ctx.normalize([static_]);
  } else {
    const messages = resolveItems(node).reduce((acm, c) => [...acm, formatMessagePart(ctx, c)], []);
    return ctx.normalize(messages);
  }
}
function formatMessagePart(ctx, node) {
  const type = resolveType(node);
  switch (type) {
    case 3: {
      return resolveValue$1(node, type);
    }
    case 9: {
      return resolveValue$1(node, type);
    }
    case 4: {
      const named = node;
      if (hasOwn(named, "k") && named.k) {
        return ctx.interpolate(ctx.named(named.k));
      }
      if (hasOwn(named, "key") && named.key) {
        return ctx.interpolate(ctx.named(named.key));
      }
      throw createUnhandleNodeError(type);
    }
    case 5: {
      const list = node;
      if (hasOwn(list, "i") && isNumber(list.i)) {
        return ctx.interpolate(ctx.list(list.i));
      }
      if (hasOwn(list, "index") && isNumber(list.index)) {
        return ctx.interpolate(ctx.list(list.index));
      }
      throw createUnhandleNodeError(type);
    }
    case 6: {
      const linked = node;
      const modifier = resolveLinkedModifier(linked);
      const key = resolveLinkedKey(linked);
      return ctx.linked(formatMessagePart(ctx, key), modifier ? formatMessagePart(ctx, modifier) : void 0, ctx.type);
    }
    case 7: {
      return resolveValue$1(node, type);
    }
    case 8: {
      return resolveValue$1(node, type);
    }
    default:
      throw new Error(`unhandled node on format message part: ${type}`);
  }
}
const defaultOnCacheKey = (message) => message;
let compileCache = create();
function baseCompile(message, options = {}) {
  let detectError = false;
  const onError = options.onError || defaultOnError;
  options.onError = (err) => {
    detectError = true;
    onError(err);
  };
  return { ...baseCompile$1(message, options), detectError };
}
const compileToFunction = /* @__NO_SIDE_EFFECTS__ */ (message, context) => {
  if (!isString$1(message)) {
    throw createCoreError(CoreErrorCodes.NOT_SUPPORT_NON_STRING_MESSAGE);
  }
  {
    isBoolean(context.warnHtmlMessage) ? context.warnHtmlMessage : true;
    const onCacheKey = context.onCacheKey || defaultOnCacheKey;
    const cacheKey = onCacheKey(message);
    const cached2 = compileCache[cacheKey];
    if (cached2) {
      return cached2;
    }
    const { code: code2, detectError } = baseCompile(message, context);
    const msg = new Function(`return ${code2}`)();
    return !detectError ? compileCache[cacheKey] = msg : msg;
  }
};
function compile(message, context) {
  if (__INTLIFY_JIT_COMPILATION__ && !__INTLIFY_DROP_MESSAGE_COMPILER__ && isString$1(message)) {
    isBoolean(context.warnHtmlMessage) ? context.warnHtmlMessage : true;
    const onCacheKey = context.onCacheKey || defaultOnCacheKey;
    const cacheKey = onCacheKey(message);
    const cached2 = compileCache[cacheKey];
    if (cached2) {
      return cached2;
    }
    const { ast, detectError } = baseCompile(message, {
      ...context,
      location: false,
      jit: true
    });
    const msg = format(ast);
    return !detectError ? compileCache[cacheKey] = msg : msg;
  } else {
    const cacheKey = message.cacheKey;
    if (cacheKey) {
      const cached2 = compileCache[cacheKey];
      if (cached2) {
        return cached2;
      }
      return compileCache[cacheKey] = format(message);
    } else {
      return format(message);
    }
  }
}
const NOOP_MESSAGE_FUNCTION = () => "";
const isMessageFunction = (val) => isFunction(val);
function translate(context, ...args) {
  const { fallbackFormat, postTranslation, unresolving, messageCompiler, fallbackLocale, messages } = context;
  const [key, options] = parseTranslateArgs(...args);
  const missingWarn = isBoolean(options.missingWarn) ? options.missingWarn : context.missingWarn;
  const fallbackWarn = isBoolean(options.fallbackWarn) ? options.fallbackWarn : context.fallbackWarn;
  const escapeParameter = isBoolean(options.escapeParameter) ? options.escapeParameter : context.escapeParameter;
  const resolvedMessage = !!options.resolvedMessage;
  const defaultMsgOrKey = isString$1(options.default) || isBoolean(options.default) ? !isBoolean(options.default) ? options.default : !messageCompiler ? () => key : key : fallbackFormat ? !messageCompiler ? () => key : key : "";
  const enableDefaultMsg = fallbackFormat || defaultMsgOrKey !== "";
  const locale = getLocale(context, options);
  escapeParameter && escapeParams(options);
  let [formatScope, targetLocale, message] = !resolvedMessage ? resolveMessageFormat(context, key, locale, fallbackLocale, fallbackWarn, missingWarn) : [
    key,
    locale,
    messages[locale] || create()
  ];
  let format2 = formatScope;
  let cacheBaseKey = key;
  if (!resolvedMessage && !(isString$1(format2) || isMessageAST(format2) || isMessageFunction(format2))) {
    if (enableDefaultMsg) {
      format2 = defaultMsgOrKey;
      cacheBaseKey = format2;
    }
  }
  if (!resolvedMessage && (!(isString$1(format2) || isMessageAST(format2) || isMessageFunction(format2)) || !isString$1(targetLocale))) {
    return unresolving ? NOT_REOSLVED : key;
  }
  let occurred = false;
  const onError = () => {
    occurred = true;
  };
  const msg = !isMessageFunction(format2) ? compileMessageFormat(context, key, targetLocale, format2, cacheBaseKey, onError) : format2;
  if (occurred) {
    return format2;
  }
  const ctxOptions = getMessageContextOptions(context, targetLocale, message, options);
  const msgContext = createMessageContext(ctxOptions);
  const messaged = evaluateMessage(context, msg, msgContext);
  let ret = postTranslation ? postTranslation(messaged, key) : messaged;
  if (escapeParameter && isString$1(ret)) {
    ret = sanitizeTranslatedHtml(ret);
  }
  if (__INTLIFY_PROD_DEVTOOLS__) {
    const payloads = {
      timestamp: Date.now(),
      key: isString$1(key) ? key : isMessageFunction(format2) ? format2.key : "",
      locale: targetLocale || (isMessageFunction(format2) ? format2.locale : ""),
      format: isString$1(format2) ? format2 : isMessageFunction(format2) ? format2.source : "",
      message: ret
    };
    payloads.meta = assign$1({}, context.__meta, /* @__PURE__ */ getAdditionalMeta() || {});
    translateDevTools(payloads);
  }
  return ret;
}
function escapeParams(options) {
  if (isArray(options.list)) {
    options.list = options.list.map((item) => isString$1(item) ? escapeHtml$1(item) : item);
  } else if (isObject$1(options.named)) {
    Object.keys(options.named).forEach((key) => {
      if (isString$1(options.named[key])) {
        options.named[key] = escapeHtml$1(options.named[key]);
      }
    });
  }
}
function resolveMessageFormat(context, key, locale, fallbackLocale, fallbackWarn, missingWarn) {
  const { messages, onWarn, messageResolver: resolveValue2, localeFallbacker } = context;
  const locales = localeFallbacker(context, fallbackLocale, locale);
  let message = create();
  let targetLocale;
  let format2 = null;
  const type = "translate";
  for (let i = 0; i < locales.length; i++) {
    targetLocale = locales[i];
    message = messages[targetLocale] || create();
    if ((format2 = resolveValue2(message, key)) === null) {
      format2 = message[key];
    }
    if (isString$1(format2) || isMessageAST(format2) || isMessageFunction(format2)) {
      break;
    }
    if (!isImplicitFallback(targetLocale, locales)) {
      const missingRet = handleMissing(
        context,
        // eslint-disable-line @typescript-eslint/no-explicit-any
        key,
        targetLocale,
        missingWarn,
        type
      );
      if (missingRet !== key) {
        format2 = missingRet;
      }
    }
  }
  return [format2, targetLocale, message];
}
function compileMessageFormat(context, key, targetLocale, format2, cacheBaseKey, onError) {
  const { messageCompiler, warnHtmlMessage } = context;
  if (isMessageFunction(format2)) {
    const msg2 = format2;
    msg2.locale = msg2.locale || targetLocale;
    msg2.key = msg2.key || key;
    return msg2;
  }
  if (messageCompiler == null) {
    const msg2 = () => format2;
    msg2.locale = targetLocale;
    msg2.key = key;
    return msg2;
  }
  const msg = messageCompiler(format2, getCompileContext(context, targetLocale, cacheBaseKey, format2, warnHtmlMessage, onError));
  msg.locale = targetLocale;
  msg.key = key;
  msg.source = format2;
  return msg;
}
function evaluateMessage(context, msg, msgCtx) {
  const messaged = msg(msgCtx);
  return messaged;
}
function parseTranslateArgs(...args) {
  const [arg1, arg2, arg3] = args;
  const options = create();
  if (!isString$1(arg1) && !isNumber(arg1) && !isMessageFunction(arg1) && !isMessageAST(arg1)) {
    throw createCoreError(CoreErrorCodes.INVALID_ARGUMENT);
  }
  const key = isNumber(arg1) ? String(arg1) : isMessageFunction(arg1) ? arg1 : arg1;
  if (isNumber(arg2)) {
    options.plural = arg2;
  } else if (isString$1(arg2)) {
    options.default = arg2;
  } else if (isPlainObject(arg2) && !isEmptyObject(arg2)) {
    options.named = arg2;
  } else if (isArray(arg2)) {
    options.list = arg2;
  }
  if (isNumber(arg3)) {
    options.plural = arg3;
  } else if (isString$1(arg3)) {
    options.default = arg3;
  } else if (isPlainObject(arg3)) {
    assign$1(options, arg3);
  }
  return [key, options];
}
function getCompileContext(context, locale, key, source, warnHtmlMessage, onError) {
  return {
    locale,
    key,
    warnHtmlMessage,
    onError: (err) => {
      onError && onError(err);
      {
        throw err;
      }
    },
    onCacheKey: (source2) => generateFormatCacheKey(locale, key, source2)
  };
}
function getMessageContextOptions(context, locale, message, options) {
  const { modifiers, pluralRules, messageResolver: resolveValue2, fallbackLocale, fallbackWarn, missingWarn, fallbackContext } = context;
  const resolveMessage = (key) => {
    let val = resolveValue2(message, key);
    if (val == null && fallbackContext) {
      const [, , message2] = resolveMessageFormat(fallbackContext, key, locale, fallbackLocale, fallbackWarn, missingWarn);
      val = resolveValue2(message2, key);
    }
    if (isString$1(val) || isMessageAST(val)) {
      let occurred = false;
      const onError = () => {
        occurred = true;
      };
      const msg = compileMessageFormat(context, key, locale, val, key, onError);
      return !occurred ? msg : NOOP_MESSAGE_FUNCTION;
    } else if (isMessageFunction(val)) {
      return val;
    } else {
      return NOOP_MESSAGE_FUNCTION;
    }
  };
  const ctxOptions = {
    locale,
    modifiers,
    pluralRules,
    messages: resolveMessage
  };
  if (context.processor) {
    ctxOptions.processor = context.processor;
  }
  if (options.list) {
    ctxOptions.list = options.list;
  }
  if (options.named) {
    ctxOptions.named = options.named;
  }
  if (isNumber(options.plural)) {
    ctxOptions.pluralIndex = options.plural;
  }
  return ctxOptions;
}
function datetime(context, ...args) {
  const { datetimeFormats, unresolving, fallbackLocale, onWarn, localeFallbacker } = context;
  const { __datetimeFormatters } = context;
  const [key, value, options, overrides] = parseDateTimeArgs(...args);
  const missingWarn = isBoolean(options.missingWarn) ? options.missingWarn : context.missingWarn;
  isBoolean(options.fallbackWarn) ? options.fallbackWarn : context.fallbackWarn;
  const part = !!options.part;
  const locale = getLocale(context, options);
  const locales = localeFallbacker(
    context,
    // eslint-disable-line @typescript-eslint/no-explicit-any
    fallbackLocale,
    locale
  );
  if (!isString$1(key) || key === "") {
    return new Intl.DateTimeFormat(locale, overrides).format(value);
  }
  let datetimeFormat = {};
  let targetLocale;
  let format2 = null;
  const type = "datetime format";
  for (let i = 0; i < locales.length; i++) {
    targetLocale = locales[i];
    datetimeFormat = datetimeFormats[targetLocale] || {};
    format2 = datetimeFormat[key];
    if (isPlainObject(format2))
      break;
    handleMissing(context, key, targetLocale, missingWarn, type);
  }
  if (!isPlainObject(format2) || !isString$1(targetLocale)) {
    return unresolving ? NOT_REOSLVED : key;
  }
  let id = `${targetLocale}__${key}`;
  if (!isEmptyObject(overrides)) {
    id = `${id}__${JSON.stringify(overrides)}`;
  }
  let formatter = __datetimeFormatters.get(id);
  if (!formatter) {
    formatter = new Intl.DateTimeFormat(targetLocale, assign$1({}, format2, overrides));
    __datetimeFormatters.set(id, formatter);
  }
  return !part ? formatter.format(value) : formatter.formatToParts(value);
}
const DATETIME_FORMAT_OPTIONS_KEYS = [
  "localeMatcher",
  "weekday",
  "era",
  "year",
  "month",
  "day",
  "hour",
  "minute",
  "second",
  "timeZoneName",
  "formatMatcher",
  "hour12",
  "timeZone",
  "dateStyle",
  "timeStyle",
  "calendar",
  "dayPeriod",
  "numberingSystem",
  "hourCycle",
  "fractionalSecondDigits"
];
function parseDateTimeArgs(...args) {
  const [arg1, arg2, arg3, arg4] = args;
  const options = create();
  let overrides = create();
  let value;
  if (isString$1(arg1)) {
    const matches = arg1.match(/(\d{4}-\d{2}-\d{2})(T|\s)?(.*)/);
    if (!matches) {
      throw createCoreError(CoreErrorCodes.INVALID_ISO_DATE_ARGUMENT);
    }
    const dateTime = matches[3] ? matches[3].trim().startsWith("T") ? `${matches[1].trim()}${matches[3].trim()}` : `${matches[1].trim()}T${matches[3].trim()}` : matches[1].trim();
    value = new Date(dateTime);
    try {
      value.toISOString();
    } catch (e) {
      throw createCoreError(CoreErrorCodes.INVALID_ISO_DATE_ARGUMENT);
    }
  } else if (isDate(arg1)) {
    if (isNaN(arg1.getTime())) {
      throw createCoreError(CoreErrorCodes.INVALID_DATE_ARGUMENT);
    }
    value = arg1;
  } else if (isNumber(arg1)) {
    value = arg1;
  } else {
    throw createCoreError(CoreErrorCodes.INVALID_ARGUMENT);
  }
  if (isString$1(arg2)) {
    options.key = arg2;
  } else if (isPlainObject(arg2)) {
    Object.keys(arg2).forEach((key) => {
      if (DATETIME_FORMAT_OPTIONS_KEYS.includes(key)) {
        overrides[key] = arg2[key];
      } else {
        options[key] = arg2[key];
      }
    });
  }
  if (isString$1(arg3)) {
    options.locale = arg3;
  } else if (isPlainObject(arg3)) {
    overrides = arg3;
  }
  if (isPlainObject(arg4)) {
    overrides = arg4;
  }
  return [options.key || "", value, options, overrides];
}
function clearDateTimeFormat(ctx, locale, format2) {
  const context = ctx;
  for (const key in format2) {
    const id = `${locale}__${key}`;
    if (!context.__datetimeFormatters.has(id)) {
      continue;
    }
    context.__datetimeFormatters.delete(id);
  }
}
function number(context, ...args) {
  const { numberFormats, unresolving, fallbackLocale, onWarn, localeFallbacker } = context;
  const { __numberFormatters } = context;
  const [key, value, options, overrides] = parseNumberArgs(...args);
  const missingWarn = isBoolean(options.missingWarn) ? options.missingWarn : context.missingWarn;
  isBoolean(options.fallbackWarn) ? options.fallbackWarn : context.fallbackWarn;
  const part = !!options.part;
  const locale = getLocale(context, options);
  const locales = localeFallbacker(
    context,
    // eslint-disable-line @typescript-eslint/no-explicit-any
    fallbackLocale,
    locale
  );
  if (!isString$1(key) || key === "") {
    return new Intl.NumberFormat(locale, overrides).format(value);
  }
  let numberFormat = {};
  let targetLocale;
  let format2 = null;
  const type = "number format";
  for (let i = 0; i < locales.length; i++) {
    targetLocale = locales[i];
    numberFormat = numberFormats[targetLocale] || {};
    format2 = numberFormat[key];
    if (isPlainObject(format2))
      break;
    handleMissing(context, key, targetLocale, missingWarn, type);
  }
  if (!isPlainObject(format2) || !isString$1(targetLocale)) {
    return unresolving ? NOT_REOSLVED : key;
  }
  let id = `${targetLocale}__${key}`;
  if (!isEmptyObject(overrides)) {
    id = `${id}__${JSON.stringify(overrides)}`;
  }
  let formatter = __numberFormatters.get(id);
  if (!formatter) {
    formatter = new Intl.NumberFormat(targetLocale, assign$1({}, format2, overrides));
    __numberFormatters.set(id, formatter);
  }
  return !part ? formatter.format(value) : formatter.formatToParts(value);
}
const NUMBER_FORMAT_OPTIONS_KEYS = [
  "localeMatcher",
  "style",
  "currency",
  "currencyDisplay",
  "currencySign",
  "useGrouping",
  "minimumIntegerDigits",
  "minimumFractionDigits",
  "maximumFractionDigits",
  "minimumSignificantDigits",
  "maximumSignificantDigits",
  "compactDisplay",
  "notation",
  "signDisplay",
  "unit",
  "unitDisplay",
  "roundingMode",
  "roundingPriority",
  "roundingIncrement",
  "trailingZeroDisplay"
];
function parseNumberArgs(...args) {
  const [arg1, arg2, arg3, arg4] = args;
  const options = create();
  let overrides = create();
  if (!isNumber(arg1)) {
    throw createCoreError(CoreErrorCodes.INVALID_ARGUMENT);
  }
  const value = arg1;
  if (isString$1(arg2)) {
    options.key = arg2;
  } else if (isPlainObject(arg2)) {
    Object.keys(arg2).forEach((key) => {
      if (NUMBER_FORMAT_OPTIONS_KEYS.includes(key)) {
        overrides[key] = arg2[key];
      } else {
        options[key] = arg2[key];
      }
    });
  }
  if (isString$1(arg3)) {
    options.locale = arg3;
  } else if (isPlainObject(arg3)) {
    overrides = arg3;
  }
  if (isPlainObject(arg4)) {
    overrides = arg4;
  }
  return [options.key || "", value, options, overrides];
}
function clearNumberFormat(ctx, locale, format2) {
  const context = ctx;
  for (const key in format2) {
    const id = `${locale}__${key}`;
    if (!context.__numberFormatters.has(id)) {
      continue;
    }
    context.__numberFormatters.delete(id);
  }
}
{
  initFeatureFlags$1();
}
/*!
  * vue-i18n v9.14.5
  * (c) 2025 kazuya kawaguchi
  * Released under the MIT License.
  */
const VERSION = "9.14.5";
function initFeatureFlags() {
  if (typeof __VUE_I18N_FULL_INSTALL__ !== "boolean") {
    getGlobalThis().__VUE_I18N_FULL_INSTALL__ = true;
  }
  if (typeof __VUE_I18N_LEGACY_API__ !== "boolean") {
    getGlobalThis().__VUE_I18N_LEGACY_API__ = true;
  }
  if (typeof __INTLIFY_JIT_COMPILATION__ !== "boolean") {
    getGlobalThis().__INTLIFY_JIT_COMPILATION__ = false;
  }
  if (typeof __INTLIFY_DROP_MESSAGE_COMPILER__ !== "boolean") {
    getGlobalThis().__INTLIFY_DROP_MESSAGE_COMPILER__ = false;
  }
  if (typeof __INTLIFY_PROD_DEVTOOLS__ !== "boolean") {
    getGlobalThis().__INTLIFY_PROD_DEVTOOLS__ = false;
  }
}
const code$1 = CoreWarnCodes.__EXTEND_POINT__;
const inc$1 = incrementer(code$1);
({
  // 9
  NOT_SUPPORTED_PRESERVE: inc$1(),
  // 10
  NOT_SUPPORTED_FORMATTER: inc$1(),
  // 11
  NOT_SUPPORTED_PRESERVE_DIRECTIVE: inc$1(),
  // 12
  NOT_SUPPORTED_GET_CHOICE_INDEX: inc$1(),
  // 13
  COMPONENT_NAME_LEGACY_COMPATIBLE: inc$1(),
  // 14
  NOT_FOUND_PARENT_SCOPE: inc$1(),
  // 15
  IGNORE_OBJ_FLATTEN: inc$1(),
  // 16
  NOTICE_DROP_ALLOW_COMPOSITION: inc$1(),
  // 17
  NOTICE_DROP_TRANSLATE_EXIST_COMPATIBLE_FLAG: inc$1()
  // 18
});
const code = CoreErrorCodes.__EXTEND_POINT__;
const inc = incrementer(code);
const I18nErrorCodes = {
  // composer module errors
  UNEXPECTED_RETURN_TYPE: code,
  // 24
  // legacy module errors
  INVALID_ARGUMENT: inc(),
  // 25
  // i18n module errors
  MUST_BE_CALL_SETUP_TOP: inc(),
  // 26
  NOT_INSTALLED: inc(),
  // 27
  NOT_AVAILABLE_IN_LEGACY_MODE: inc(),
  // 28
  // directive module errors
  REQUIRED_VALUE: inc(),
  // 29
  INVALID_VALUE: inc(),
  // 30
  // vue-devtools errors
  CANNOT_SETUP_VUE_DEVTOOLS_PLUGIN: inc(),
  // 31
  NOT_INSTALLED_WITH_PROVIDE: inc(),
  // 32
  // unexpected error
  UNEXPECTED_ERROR: inc(),
  // 33
  // not compatible legacy vue-i18n constructor
  NOT_COMPATIBLE_LEGACY_VUE_I18N: inc(),
  // 34
  // bridge support vue 2.x only
  BRIDGE_SUPPORT_VUE_2_ONLY: inc(),
  // 35
  // need to define `i18n` option in `allowComposition: true` and `useScope: 'local' at `useI18n``
  MUST_DEFINE_I18N_OPTION_IN_ALLOW_COMPOSITION: inc(),
  // 36
  // Not available Compostion API in Legacy API mode. Please make sure that the legacy API mode is working properly
  NOT_AVAILABLE_COMPOSITION_IN_LEGACY: inc(),
  // 37
  // for enhancement
  __EXTEND_POINT__: inc()
  // 38
};
function createI18nError(code2, ...args) {
  return createCompileError(code2, null, void 0);
}
const TranslateVNodeSymbol = /* @__PURE__ */ makeSymbol("__translateVNode");
const DatetimePartsSymbol = /* @__PURE__ */ makeSymbol("__datetimeParts");
const NumberPartsSymbol = /* @__PURE__ */ makeSymbol("__numberParts");
const SetPluralRulesSymbol = makeSymbol("__setPluralRules");
const InejctWithOptionSymbol = /* @__PURE__ */ makeSymbol("__injectWithOption");
const DisposeSymbol = /* @__PURE__ */ makeSymbol("__dispose");
function handleFlatJson(obj) {
  if (!isObject$1(obj)) {
    return obj;
  }
  if (isMessageAST(obj)) {
    return obj;
  }
  for (const key in obj) {
    if (!hasOwn(obj, key)) {
      continue;
    }
    if (!key.includes(".")) {
      if (isObject$1(obj[key])) {
        handleFlatJson(obj[key]);
      }
    } else {
      const subKeys = key.split(".");
      const lastIndex = subKeys.length - 1;
      let currentObj = obj;
      let hasStringValue = false;
      for (let i = 0; i < lastIndex; i++) {
        if (subKeys[i] === "__proto__") {
          throw new Error(`unsafe key: ${subKeys[i]}`);
        }
        if (!(subKeys[i] in currentObj)) {
          currentObj[subKeys[i]] = create();
        }
        if (!isObject$1(currentObj[subKeys[i]])) {
          hasStringValue = true;
          break;
        }
        currentObj = currentObj[subKeys[i]];
      }
      if (!hasStringValue) {
        if (!isMessageAST(currentObj)) {
          currentObj[subKeys[lastIndex]] = obj[key];
          delete obj[key];
        } else {
          if (!AST_NODE_PROPS_KEYS.includes(subKeys[lastIndex])) {
            delete obj[key];
          }
        }
      }
      if (!isMessageAST(currentObj)) {
        const target = currentObj[subKeys[lastIndex]];
        if (isObject$1(target)) {
          handleFlatJson(target);
        }
      }
    }
  }
  return obj;
}
function getLocaleMessages(locale, options) {
  const { messages, __i18n, messageResolver, flatJson } = options;
  const ret = isPlainObject(messages) ? messages : isArray(__i18n) ? create() : { [locale]: create() };
  if (isArray(__i18n)) {
    __i18n.forEach((custom) => {
      if ("locale" in custom && "resource" in custom) {
        const { locale: locale2, resource } = custom;
        if (locale2) {
          ret[locale2] = ret[locale2] || create();
          deepCopy(resource, ret[locale2]);
        } else {
          deepCopy(resource, ret);
        }
      } else {
        isString$1(custom) && deepCopy(JSON.parse(custom), ret);
      }
    });
  }
  if (messageResolver == null && flatJson) {
    for (const key in ret) {
      if (hasOwn(ret, key)) {
        handleFlatJson(ret[key]);
      }
    }
  }
  return ret;
}
function getComponentOptions(instance) {
  return instance.type;
}
function adjustI18nResources(gl, options, componentOptions) {
  let messages = isObject$1(options.messages) ? options.messages : create();
  if ("__i18nGlobal" in componentOptions) {
    messages = getLocaleMessages(gl.locale.value, {
      messages,
      __i18n: componentOptions.__i18nGlobal
    });
  }
  const locales = Object.keys(messages);
  if (locales.length) {
    locales.forEach((locale) => {
      gl.mergeLocaleMessage(locale, messages[locale]);
    });
  }
  {
    if (isObject$1(options.datetimeFormats)) {
      const locales2 = Object.keys(options.datetimeFormats);
      if (locales2.length) {
        locales2.forEach((locale) => {
          gl.mergeDateTimeFormat(locale, options.datetimeFormats[locale]);
        });
      }
    }
    if (isObject$1(options.numberFormats)) {
      const locales2 = Object.keys(options.numberFormats);
      if (locales2.length) {
        locales2.forEach((locale) => {
          gl.mergeNumberFormat(locale, options.numberFormats[locale]);
        });
      }
    }
  }
}
function createTextNode(key) {
  return createVNode(Text, null, key, 0);
}
const DEVTOOLS_META = "__INTLIFY_META__";
const NOOP_RETURN_ARRAY = () => [];
const NOOP_RETURN_FALSE = () => false;
let composerID = 0;
function defineCoreMissingHandler(missing) {
  return (ctx, locale, key, type) => {
    return missing(locale, key, getCurrentInstance() || void 0, type);
  };
}
const getMetaInfo = /* @__NO_SIDE_EFFECTS__ */ () => {
  const instance = getCurrentInstance();
  let meta = null;
  return instance && (meta = getComponentOptions(instance)[DEVTOOLS_META]) ? { [DEVTOOLS_META]: meta } : null;
};
function createComposer(options = {}, VueI18nLegacy) {
  const { __root, __injectWithOption } = options;
  const _isGlobal = __root === void 0;
  const flatJson = options.flatJson;
  const _ref = inBrowser ? ref : shallowRef;
  const translateExistCompatible = !!options.translateExistCompatible;
  let _inheritLocale = isBoolean(options.inheritLocale) ? options.inheritLocale : true;
  const _locale = _ref(
    // prettier-ignore
    __root && _inheritLocale ? __root.locale.value : isString$1(options.locale) ? options.locale : DEFAULT_LOCALE
  );
  const _fallbackLocale = _ref(
    // prettier-ignore
    __root && _inheritLocale ? __root.fallbackLocale.value : isString$1(options.fallbackLocale) || isArray(options.fallbackLocale) || isPlainObject(options.fallbackLocale) || options.fallbackLocale === false ? options.fallbackLocale : _locale.value
  );
  const _messages = _ref(getLocaleMessages(_locale.value, options));
  const _datetimeFormats = _ref(isPlainObject(options.datetimeFormats) ? options.datetimeFormats : { [_locale.value]: {} });
  const _numberFormats = _ref(isPlainObject(options.numberFormats) ? options.numberFormats : { [_locale.value]: {} });
  let _missingWarn = __root ? __root.missingWarn : isBoolean(options.missingWarn) || isRegExp(options.missingWarn) ? options.missingWarn : true;
  let _fallbackWarn = __root ? __root.fallbackWarn : isBoolean(options.fallbackWarn) || isRegExp(options.fallbackWarn) ? options.fallbackWarn : true;
  let _fallbackRoot = __root ? __root.fallbackRoot : isBoolean(options.fallbackRoot) ? options.fallbackRoot : true;
  let _fallbackFormat = !!options.fallbackFormat;
  let _missing = isFunction(options.missing) ? options.missing : null;
  let _runtimeMissing = isFunction(options.missing) ? defineCoreMissingHandler(options.missing) : null;
  let _postTranslation = isFunction(options.postTranslation) ? options.postTranslation : null;
  let _warnHtmlMessage = __root ? __root.warnHtmlMessage : isBoolean(options.warnHtmlMessage) ? options.warnHtmlMessage : true;
  let _escapeParameter = !!options.escapeParameter;
  const _modifiers = __root ? __root.modifiers : isPlainObject(options.modifiers) ? options.modifiers : {};
  let _pluralRules = options.pluralRules || __root && __root.pluralRules;
  let _context;
  const getCoreContext = () => {
    _isGlobal && setFallbackContext(null);
    const ctxOptions = {
      version: VERSION,
      locale: _locale.value,
      fallbackLocale: _fallbackLocale.value,
      messages: _messages.value,
      modifiers: _modifiers,
      pluralRules: _pluralRules,
      missing: _runtimeMissing === null ? void 0 : _runtimeMissing,
      missingWarn: _missingWarn,
      fallbackWarn: _fallbackWarn,
      fallbackFormat: _fallbackFormat,
      unresolving: true,
      postTranslation: _postTranslation === null ? void 0 : _postTranslation,
      warnHtmlMessage: _warnHtmlMessage,
      escapeParameter: _escapeParameter,
      messageResolver: options.messageResolver,
      messageCompiler: options.messageCompiler,
      __meta: { framework: "vue" }
    };
    {
      ctxOptions.datetimeFormats = _datetimeFormats.value;
      ctxOptions.numberFormats = _numberFormats.value;
      ctxOptions.__datetimeFormatters = isPlainObject(_context) ? _context.__datetimeFormatters : void 0;
      ctxOptions.__numberFormatters = isPlainObject(_context) ? _context.__numberFormatters : void 0;
    }
    const ctx = createCoreContext(ctxOptions);
    _isGlobal && setFallbackContext(ctx);
    return ctx;
  };
  _context = getCoreContext();
  updateFallbackLocale(_context, _locale.value, _fallbackLocale.value);
  function trackReactivityValues() {
    return [
      _locale.value,
      _fallbackLocale.value,
      _messages.value,
      _datetimeFormats.value,
      _numberFormats.value
    ];
  }
  const locale = computed({
    get: () => _locale.value,
    set: (val) => {
      _locale.value = val;
      _context.locale = _locale.value;
    }
  });
  const fallbackLocale = computed({
    get: () => _fallbackLocale.value,
    set: (val) => {
      _fallbackLocale.value = val;
      _context.fallbackLocale = _fallbackLocale.value;
      updateFallbackLocale(_context, _locale.value, val);
    }
  });
  const messages = computed(() => _messages.value);
  const datetimeFormats = /* @__PURE__ */ computed(() => _datetimeFormats.value);
  const numberFormats = /* @__PURE__ */ computed(() => _numberFormats.value);
  function getPostTranslationHandler() {
    return isFunction(_postTranslation) ? _postTranslation : null;
  }
  function setPostTranslationHandler(handler) {
    _postTranslation = handler;
    _context.postTranslation = handler;
  }
  function getMissingHandler() {
    return _missing;
  }
  function setMissingHandler(handler) {
    if (handler !== null) {
      _runtimeMissing = defineCoreMissingHandler(handler);
    }
    _missing = handler;
    _context.missing = _runtimeMissing;
  }
  const wrapWithDeps = (fn, argumentParser, warnType, fallbackSuccess, fallbackFail, successCondition) => {
    trackReactivityValues();
    let ret;
    try {
      if (__INTLIFY_PROD_DEVTOOLS__) {
        /* @__PURE__ */ setAdditionalMeta(/* @__PURE__ */ getMetaInfo());
      }
      if (!_isGlobal) {
        _context.fallbackContext = __root ? getFallbackContext() : void 0;
      }
      ret = fn(_context);
    } finally {
      if (__INTLIFY_PROD_DEVTOOLS__) ;
      if (!_isGlobal) {
        _context.fallbackContext = void 0;
      }
    }
    if (warnType !== "translate exists" && // for not `te` (e.g `t`)
    isNumber(ret) && ret === NOT_REOSLVED || warnType === "translate exists" && !ret) {
      const [key, arg2] = argumentParser();
      return __root && _fallbackRoot ? fallbackSuccess(__root) : fallbackFail(key);
    } else if (successCondition(ret)) {
      return ret;
    } else {
      throw createI18nError(I18nErrorCodes.UNEXPECTED_RETURN_TYPE);
    }
  };
  function t(...args) {
    return wrapWithDeps((context) => Reflect.apply(translate, null, [context, ...args]), () => parseTranslateArgs(...args), "translate", (root) => Reflect.apply(root.t, root, [...args]), (key) => key, (val) => isString$1(val));
  }
  function rt(...args) {
    const [arg1, arg2, arg3] = args;
    if (arg3 && !isObject$1(arg3)) {
      throw createI18nError(I18nErrorCodes.INVALID_ARGUMENT);
    }
    return t(...[arg1, arg2, assign$1({ resolvedMessage: true }, arg3 || {})]);
  }
  function d(...args) {
    return wrapWithDeps((context) => Reflect.apply(datetime, null, [context, ...args]), () => parseDateTimeArgs(...args), "datetime format", (root) => Reflect.apply(root.d, root, [...args]), () => MISSING_RESOLVE_VALUE, (val) => isString$1(val));
  }
  function n(...args) {
    return wrapWithDeps((context) => Reflect.apply(number, null, [context, ...args]), () => parseNumberArgs(...args), "number format", (root) => Reflect.apply(root.n, root, [...args]), () => MISSING_RESOLVE_VALUE, (val) => isString$1(val));
  }
  function normalize(values) {
    return values.map((val) => isString$1(val) || isNumber(val) || isBoolean(val) ? createTextNode(String(val)) : val);
  }
  const interpolate = (val) => val;
  const processor = {
    normalize,
    interpolate,
    type: "vnode"
  };
  function translateVNode(...args) {
    return wrapWithDeps(
      (context) => {
        let ret;
        const _context2 = context;
        try {
          _context2.processor = processor;
          ret = Reflect.apply(translate, null, [_context2, ...args]);
        } finally {
          _context2.processor = null;
        }
        return ret;
      },
      () => parseTranslateArgs(...args),
      "translate",
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (root) => root[TranslateVNodeSymbol](...args),
      (key) => [createTextNode(key)],
      (val) => isArray(val)
    );
  }
  function numberParts(...args) {
    return wrapWithDeps(
      (context) => Reflect.apply(number, null, [context, ...args]),
      () => parseNumberArgs(...args),
      "number format",
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (root) => root[NumberPartsSymbol](...args),
      NOOP_RETURN_ARRAY,
      (val) => isString$1(val) || isArray(val)
    );
  }
  function datetimeParts(...args) {
    return wrapWithDeps(
      (context) => Reflect.apply(datetime, null, [context, ...args]),
      () => parseDateTimeArgs(...args),
      "datetime format",
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (root) => root[DatetimePartsSymbol](...args),
      NOOP_RETURN_ARRAY,
      (val) => isString$1(val) || isArray(val)
    );
  }
  function setPluralRules(rules) {
    _pluralRules = rules;
    _context.pluralRules = _pluralRules;
  }
  function te(key, locale2) {
    return wrapWithDeps(() => {
      if (!key) {
        return false;
      }
      const targetLocale = isString$1(locale2) ? locale2 : _locale.value;
      const message = getLocaleMessage(targetLocale);
      const resolved = _context.messageResolver(message, key);
      return !translateExistCompatible ? isMessageAST(resolved) || isMessageFunction(resolved) || isString$1(resolved) : resolved != null;
    }, () => [key], "translate exists", (root) => {
      return Reflect.apply(root.te, root, [key, locale2]);
    }, NOOP_RETURN_FALSE, (val) => isBoolean(val));
  }
  function resolveMessages(key) {
    let messages2 = null;
    const locales = fallbackWithLocaleChain(_context, _fallbackLocale.value, _locale.value);
    for (let i = 0; i < locales.length; i++) {
      const targetLocaleMessages = _messages.value[locales[i]] || {};
      const messageValue = _context.messageResolver(targetLocaleMessages, key);
      if (messageValue != null) {
        messages2 = messageValue;
        break;
      }
    }
    return messages2;
  }
  function tm(key) {
    const messages2 = resolveMessages(key);
    return messages2 != null ? messages2 : __root ? __root.tm(key) || {} : {};
  }
  function getLocaleMessage(locale2) {
    return _messages.value[locale2] || {};
  }
  function setLocaleMessage(locale2, message) {
    if (flatJson) {
      const _message = { [locale2]: message };
      for (const key in _message) {
        if (hasOwn(_message, key)) {
          handleFlatJson(_message[key]);
        }
      }
      message = _message[locale2];
    }
    _messages.value[locale2] = message;
    _context.messages = _messages.value;
  }
  function mergeLocaleMessage(locale2, message) {
    _messages.value[locale2] = _messages.value[locale2] || {};
    const _message = { [locale2]: message };
    if (flatJson) {
      for (const key in _message) {
        if (hasOwn(_message, key)) {
          handleFlatJson(_message[key]);
        }
      }
    }
    message = _message[locale2];
    deepCopy(message, _messages.value[locale2]);
    _context.messages = _messages.value;
  }
  function getDateTimeFormat(locale2) {
    return _datetimeFormats.value[locale2] || {};
  }
  function setDateTimeFormat(locale2, format2) {
    _datetimeFormats.value[locale2] = format2;
    _context.datetimeFormats = _datetimeFormats.value;
    clearDateTimeFormat(_context, locale2, format2);
  }
  function mergeDateTimeFormat(locale2, format2) {
    _datetimeFormats.value[locale2] = assign$1(_datetimeFormats.value[locale2] || {}, format2);
    _context.datetimeFormats = _datetimeFormats.value;
    clearDateTimeFormat(_context, locale2, format2);
  }
  function getNumberFormat(locale2) {
    return _numberFormats.value[locale2] || {};
  }
  function setNumberFormat(locale2, format2) {
    _numberFormats.value[locale2] = format2;
    _context.numberFormats = _numberFormats.value;
    clearNumberFormat(_context, locale2, format2);
  }
  function mergeNumberFormat(locale2, format2) {
    _numberFormats.value[locale2] = assign$1(_numberFormats.value[locale2] || {}, format2);
    _context.numberFormats = _numberFormats.value;
    clearNumberFormat(_context, locale2, format2);
  }
  composerID++;
  if (__root && inBrowser) {
    watch(__root.locale, (val) => {
      if (_inheritLocale) {
        _locale.value = val;
        _context.locale = val;
        updateFallbackLocale(_context, _locale.value, _fallbackLocale.value);
      }
    });
    watch(__root.fallbackLocale, (val) => {
      if (_inheritLocale) {
        _fallbackLocale.value = val;
        _context.fallbackLocale = val;
        updateFallbackLocale(_context, _locale.value, _fallbackLocale.value);
      }
    });
  }
  const composer = {
    id: composerID,
    locale,
    fallbackLocale,
    get inheritLocale() {
      return _inheritLocale;
    },
    set inheritLocale(val) {
      _inheritLocale = val;
      if (val && __root) {
        _locale.value = __root.locale.value;
        _fallbackLocale.value = __root.fallbackLocale.value;
        updateFallbackLocale(_context, _locale.value, _fallbackLocale.value);
      }
    },
    get availableLocales() {
      return Object.keys(_messages.value).sort();
    },
    messages,
    get modifiers() {
      return _modifiers;
    },
    get pluralRules() {
      return _pluralRules || {};
    },
    get isGlobal() {
      return _isGlobal;
    },
    get missingWarn() {
      return _missingWarn;
    },
    set missingWarn(val) {
      _missingWarn = val;
      _context.missingWarn = _missingWarn;
    },
    get fallbackWarn() {
      return _fallbackWarn;
    },
    set fallbackWarn(val) {
      _fallbackWarn = val;
      _context.fallbackWarn = _fallbackWarn;
    },
    get fallbackRoot() {
      return _fallbackRoot;
    },
    set fallbackRoot(val) {
      _fallbackRoot = val;
    },
    get fallbackFormat() {
      return _fallbackFormat;
    },
    set fallbackFormat(val) {
      _fallbackFormat = val;
      _context.fallbackFormat = _fallbackFormat;
    },
    get warnHtmlMessage() {
      return _warnHtmlMessage;
    },
    set warnHtmlMessage(val) {
      _warnHtmlMessage = val;
      _context.warnHtmlMessage = val;
    },
    get escapeParameter() {
      return _escapeParameter;
    },
    set escapeParameter(val) {
      _escapeParameter = val;
      _context.escapeParameter = val;
    },
    t,
    getLocaleMessage,
    setLocaleMessage,
    mergeLocaleMessage,
    getPostTranslationHandler,
    setPostTranslationHandler,
    getMissingHandler,
    setMissingHandler,
    [SetPluralRulesSymbol]: setPluralRules
  };
  {
    composer.datetimeFormats = datetimeFormats;
    composer.numberFormats = numberFormats;
    composer.rt = rt;
    composer.te = te;
    composer.tm = tm;
    composer.d = d;
    composer.n = n;
    composer.getDateTimeFormat = getDateTimeFormat;
    composer.setDateTimeFormat = setDateTimeFormat;
    composer.mergeDateTimeFormat = mergeDateTimeFormat;
    composer.getNumberFormat = getNumberFormat;
    composer.setNumberFormat = setNumberFormat;
    composer.mergeNumberFormat = mergeNumberFormat;
    composer[InejctWithOptionSymbol] = __injectWithOption;
    composer[TranslateVNodeSymbol] = translateVNode;
    composer[DatetimePartsSymbol] = datetimeParts;
    composer[NumberPartsSymbol] = numberParts;
  }
  return composer;
}
function convertComposerOptions(options) {
  const locale = isString$1(options.locale) ? options.locale : DEFAULT_LOCALE;
  const fallbackLocale = isString$1(options.fallbackLocale) || isArray(options.fallbackLocale) || isPlainObject(options.fallbackLocale) || options.fallbackLocale === false ? options.fallbackLocale : locale;
  const missing = isFunction(options.missing) ? options.missing : void 0;
  const missingWarn = isBoolean(options.silentTranslationWarn) || isRegExp(options.silentTranslationWarn) ? !options.silentTranslationWarn : true;
  const fallbackWarn = isBoolean(options.silentFallbackWarn) || isRegExp(options.silentFallbackWarn) ? !options.silentFallbackWarn : true;
  const fallbackRoot = isBoolean(options.fallbackRoot) ? options.fallbackRoot : true;
  const fallbackFormat = !!options.formatFallbackMessages;
  const modifiers = isPlainObject(options.modifiers) ? options.modifiers : {};
  const pluralizationRules = options.pluralizationRules;
  const postTranslation = isFunction(options.postTranslation) ? options.postTranslation : void 0;
  const warnHtmlMessage = isString$1(options.warnHtmlInMessage) ? options.warnHtmlInMessage !== "off" : true;
  const escapeParameter = !!options.escapeParameterHtml;
  const inheritLocale = isBoolean(options.sync) ? options.sync : true;
  let messages = options.messages;
  if (isPlainObject(options.sharedMessages)) {
    const sharedMessages = options.sharedMessages;
    const locales = Object.keys(sharedMessages);
    messages = locales.reduce((messages2, locale2) => {
      const message = messages2[locale2] || (messages2[locale2] = {});
      assign$1(message, sharedMessages[locale2]);
      return messages2;
    }, messages || {});
  }
  const { __i18n, __root, __injectWithOption } = options;
  const datetimeFormats = options.datetimeFormats;
  const numberFormats = options.numberFormats;
  const flatJson = options.flatJson;
  const translateExistCompatible = options.translateExistCompatible;
  return {
    locale,
    fallbackLocale,
    messages,
    flatJson,
    datetimeFormats,
    numberFormats,
    missing,
    missingWarn,
    fallbackWarn,
    fallbackRoot,
    fallbackFormat,
    modifiers,
    pluralRules: pluralizationRules,
    postTranslation,
    warnHtmlMessage,
    escapeParameter,
    messageResolver: options.messageResolver,
    inheritLocale,
    translateExistCompatible,
    __i18n,
    __root,
    __injectWithOption
  };
}
function createVueI18n(options = {}, VueI18nLegacy) {
  {
    const composer = createComposer(convertComposerOptions(options));
    const { __extender } = options;
    const vueI18n = {
      // id
      id: composer.id,
      // locale
      get locale() {
        return composer.locale.value;
      },
      set locale(val) {
        composer.locale.value = val;
      },
      // fallbackLocale
      get fallbackLocale() {
        return composer.fallbackLocale.value;
      },
      set fallbackLocale(val) {
        composer.fallbackLocale.value = val;
      },
      // messages
      get messages() {
        return composer.messages.value;
      },
      // datetimeFormats
      get datetimeFormats() {
        return composer.datetimeFormats.value;
      },
      // numberFormats
      get numberFormats() {
        return composer.numberFormats.value;
      },
      // availableLocales
      get availableLocales() {
        return composer.availableLocales;
      },
      // formatter
      get formatter() {
        return {
          interpolate() {
            return [];
          }
        };
      },
      set formatter(val) {
      },
      // missing
      get missing() {
        return composer.getMissingHandler();
      },
      set missing(handler) {
        composer.setMissingHandler(handler);
      },
      // silentTranslationWarn
      get silentTranslationWarn() {
        return isBoolean(composer.missingWarn) ? !composer.missingWarn : composer.missingWarn;
      },
      set silentTranslationWarn(val) {
        composer.missingWarn = isBoolean(val) ? !val : val;
      },
      // silentFallbackWarn
      get silentFallbackWarn() {
        return isBoolean(composer.fallbackWarn) ? !composer.fallbackWarn : composer.fallbackWarn;
      },
      set silentFallbackWarn(val) {
        composer.fallbackWarn = isBoolean(val) ? !val : val;
      },
      // modifiers
      get modifiers() {
        return composer.modifiers;
      },
      // formatFallbackMessages
      get formatFallbackMessages() {
        return composer.fallbackFormat;
      },
      set formatFallbackMessages(val) {
        composer.fallbackFormat = val;
      },
      // postTranslation
      get postTranslation() {
        return composer.getPostTranslationHandler();
      },
      set postTranslation(handler) {
        composer.setPostTranslationHandler(handler);
      },
      // sync
      get sync() {
        return composer.inheritLocale;
      },
      set sync(val) {
        composer.inheritLocale = val;
      },
      // warnInHtmlMessage
      get warnHtmlInMessage() {
        return composer.warnHtmlMessage ? "warn" : "off";
      },
      set warnHtmlInMessage(val) {
        composer.warnHtmlMessage = val !== "off";
      },
      // escapeParameterHtml
      get escapeParameterHtml() {
        return composer.escapeParameter;
      },
      set escapeParameterHtml(val) {
        composer.escapeParameter = val;
      },
      // preserveDirectiveContent
      get preserveDirectiveContent() {
        return true;
      },
      set preserveDirectiveContent(val) {
      },
      // pluralizationRules
      get pluralizationRules() {
        return composer.pluralRules || {};
      },
      // for internal
      __composer: composer,
      // t
      t(...args) {
        const [arg1, arg2, arg3] = args;
        const options2 = {};
        let list = null;
        let named = null;
        if (!isString$1(arg1)) {
          throw createI18nError(I18nErrorCodes.INVALID_ARGUMENT);
        }
        const key = arg1;
        if (isString$1(arg2)) {
          options2.locale = arg2;
        } else if (isArray(arg2)) {
          list = arg2;
        } else if (isPlainObject(arg2)) {
          named = arg2;
        }
        if (isArray(arg3)) {
          list = arg3;
        } else if (isPlainObject(arg3)) {
          named = arg3;
        }
        return Reflect.apply(composer.t, composer, [
          key,
          list || named || {},
          options2
        ]);
      },
      rt(...args) {
        return Reflect.apply(composer.rt, composer, [...args]);
      },
      // tc
      tc(...args) {
        const [arg1, arg2, arg3] = args;
        const options2 = { plural: 1 };
        let list = null;
        let named = null;
        if (!isString$1(arg1)) {
          throw createI18nError(I18nErrorCodes.INVALID_ARGUMENT);
        }
        const key = arg1;
        if (isString$1(arg2)) {
          options2.locale = arg2;
        } else if (isNumber(arg2)) {
          options2.plural = arg2;
        } else if (isArray(arg2)) {
          list = arg2;
        } else if (isPlainObject(arg2)) {
          named = arg2;
        }
        if (isString$1(arg3)) {
          options2.locale = arg3;
        } else if (isArray(arg3)) {
          list = arg3;
        } else if (isPlainObject(arg3)) {
          named = arg3;
        }
        return Reflect.apply(composer.t, composer, [
          key,
          list || named || {},
          options2
        ]);
      },
      // te
      te(key, locale) {
        return composer.te(key, locale);
      },
      // tm
      tm(key) {
        return composer.tm(key);
      },
      // getLocaleMessage
      getLocaleMessage(locale) {
        return composer.getLocaleMessage(locale);
      },
      // setLocaleMessage
      setLocaleMessage(locale, message) {
        composer.setLocaleMessage(locale, message);
      },
      // mergeLocaleMessage
      mergeLocaleMessage(locale, message) {
        composer.mergeLocaleMessage(locale, message);
      },
      // d
      d(...args) {
        return Reflect.apply(composer.d, composer, [...args]);
      },
      // getDateTimeFormat
      getDateTimeFormat(locale) {
        return composer.getDateTimeFormat(locale);
      },
      // setDateTimeFormat
      setDateTimeFormat(locale, format2) {
        composer.setDateTimeFormat(locale, format2);
      },
      // mergeDateTimeFormat
      mergeDateTimeFormat(locale, format2) {
        composer.mergeDateTimeFormat(locale, format2);
      },
      // n
      n(...args) {
        return Reflect.apply(composer.n, composer, [...args]);
      },
      // getNumberFormat
      getNumberFormat(locale) {
        return composer.getNumberFormat(locale);
      },
      // setNumberFormat
      setNumberFormat(locale, format2) {
        composer.setNumberFormat(locale, format2);
      },
      // mergeNumberFormat
      mergeNumberFormat(locale, format2) {
        composer.mergeNumberFormat(locale, format2);
      },
      // getChoiceIndex
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      getChoiceIndex(choice, choicesLength) {
        return -1;
      }
    };
    vueI18n.__extender = __extender;
    return vueI18n;
  }
}
const baseFormatProps = {
  tag: {
    type: [String, Object]
  },
  locale: {
    type: String
  },
  scope: {
    type: String,
    // NOTE: avoid https://github.com/microsoft/rushstack/issues/1050
    validator: (val) => val === "parent" || val === "global",
    default: "parent"
    /* ComponentI18nScope */
  },
  i18n: {
    type: Object
  }
};
function getInterpolateArg({ slots }, keys) {
  if (keys.length === 1 && keys[0] === "default") {
    const ret = slots.default ? slots.default() : [];
    return ret.reduce((slot, current) => {
      return [
        ...slot,
        // prettier-ignore
        ...current.type === Fragment ? current.children : [current]
      ];
    }, []);
  } else {
    return keys.reduce((arg, key) => {
      const slot = slots[key];
      if (slot) {
        arg[key] = slot();
      }
      return arg;
    }, create());
  }
}
function getFragmentableTag(tag) {
  return Fragment;
}
const TranslationImpl = /* @__PURE__ */ defineComponent({
  /* eslint-disable */
  name: "i18n-t",
  props: assign$1({
    keypath: {
      type: String,
      required: true
    },
    plural: {
      type: [Number, String],
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      validator: (val) => isNumber(val) || !isNaN(val)
    }
  }, baseFormatProps),
  /* eslint-enable */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  setup(props, context) {
    const { slots, attrs } = context;
    const i18n2 = props.i18n || useI18n({
      useScope: props.scope,
      __useComponent: true
    });
    return () => {
      const keys = Object.keys(slots).filter((key) => key !== "_");
      const options = create();
      if (props.locale) {
        options.locale = props.locale;
      }
      if (props.plural !== void 0) {
        options.plural = isString$1(props.plural) ? +props.plural : props.plural;
      }
      const arg = getInterpolateArg(context, keys);
      const children = i18n2[TranslateVNodeSymbol](props.keypath, arg, options);
      const assignedAttrs = assign$1(create(), attrs);
      const tag = isString$1(props.tag) || isObject$1(props.tag) ? props.tag : getFragmentableTag();
      return h(tag, assignedAttrs, children);
    };
  }
});
const Translation = TranslationImpl;
function isVNode(target) {
  return isArray(target) && !isString$1(target[0]);
}
function renderFormatter(props, context, slotKeys, partFormatter) {
  const { slots, attrs } = context;
  return () => {
    const options = { part: true };
    let overrides = create();
    if (props.locale) {
      options.locale = props.locale;
    }
    if (isString$1(props.format)) {
      options.key = props.format;
    } else if (isObject$1(props.format)) {
      if (isString$1(props.format.key)) {
        options.key = props.format.key;
      }
      overrides = Object.keys(props.format).reduce((options2, prop) => {
        return slotKeys.includes(prop) ? assign$1(create(), options2, { [prop]: props.format[prop] }) : options2;
      }, create());
    }
    const parts = partFormatter(...[props.value, options, overrides]);
    let children = [options.key];
    if (isArray(parts)) {
      children = parts.map((part, index) => {
        const slot = slots[part.type];
        const node = slot ? slot({ [part.type]: part.value, index, parts }) : [part.value];
        if (isVNode(node)) {
          node[0].key = `${part.type}-${index}`;
        }
        return node;
      });
    } else if (isString$1(parts)) {
      children = [parts];
    }
    const assignedAttrs = assign$1(create(), attrs);
    const tag = isString$1(props.tag) || isObject$1(props.tag) ? props.tag : getFragmentableTag();
    return h(tag, assignedAttrs, children);
  };
}
const NumberFormatImpl = /* @__PURE__ */ defineComponent({
  /* eslint-disable */
  name: "i18n-n",
  props: assign$1({
    value: {
      type: Number,
      required: true
    },
    format: {
      type: [String, Object]
    }
  }, baseFormatProps),
  /* eslint-enable */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  setup(props, context) {
    const i18n2 = props.i18n || useI18n({
      useScope: props.scope,
      __useComponent: true
    });
    return renderFormatter(props, context, NUMBER_FORMAT_OPTIONS_KEYS, (...args) => (
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      i18n2[NumberPartsSymbol](...args)
    ));
  }
});
const NumberFormat = NumberFormatImpl;
const DatetimeFormatImpl = /* @__PURE__ */ defineComponent({
  /* eslint-disable */
  name: "i18n-d",
  props: assign$1({
    value: {
      type: [Number, Date],
      required: true
    },
    format: {
      type: [String, Object]
    }
  }, baseFormatProps),
  /* eslint-enable */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  setup(props, context) {
    const i18n2 = props.i18n || useI18n({
      useScope: props.scope,
      __useComponent: true
    });
    return renderFormatter(props, context, DATETIME_FORMAT_OPTIONS_KEYS, (...args) => (
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      i18n2[DatetimePartsSymbol](...args)
    ));
  }
});
const DatetimeFormat = DatetimeFormatImpl;
function getComposer$2(i18n2, instance) {
  const i18nInternal = i18n2;
  if (i18n2.mode === "composition") {
    return i18nInternal.__getInstance(instance) || i18n2.global;
  } else {
    const vueI18n = i18nInternal.__getInstance(instance);
    return vueI18n != null ? vueI18n.__composer : i18n2.global.__composer;
  }
}
function vTDirective(i18n2) {
  const _process = (binding) => {
    const { instance, modifiers, value } = binding;
    if (!instance || !instance.$) {
      throw createI18nError(I18nErrorCodes.UNEXPECTED_ERROR);
    }
    const composer = getComposer$2(i18n2, instance.$);
    const parsedValue = parseValue(value);
    return [
      Reflect.apply(composer.t, composer, [...makeParams(parsedValue)]),
      composer
    ];
  };
  const register = (el, binding) => {
    const [textContent, composer] = _process(binding);
    if (inBrowser && i18n2.global === composer) {
      el.__i18nWatcher = watch(composer.locale, () => {
        binding.instance && binding.instance.$forceUpdate();
      });
    }
    el.__composer = composer;
    el.textContent = textContent;
  };
  const unregister = (el) => {
    if (inBrowser && el.__i18nWatcher) {
      el.__i18nWatcher();
      el.__i18nWatcher = void 0;
      delete el.__i18nWatcher;
    }
    if (el.__composer) {
      el.__composer = void 0;
      delete el.__composer;
    }
  };
  const update = (el, { value }) => {
    if (el.__composer) {
      const composer = el.__composer;
      const parsedValue = parseValue(value);
      el.textContent = Reflect.apply(composer.t, composer, [
        ...makeParams(parsedValue)
      ]);
    }
  };
  const getSSRProps = (binding) => {
    const [textContent] = _process(binding);
    return { textContent };
  };
  return {
    created: register,
    unmounted: unregister,
    beforeUpdate: update,
    getSSRProps
  };
}
function parseValue(value) {
  if (isString$1(value)) {
    return { path: value };
  } else if (isPlainObject(value)) {
    if (!("path" in value)) {
      throw createI18nError(I18nErrorCodes.REQUIRED_VALUE, "path");
    }
    return value;
  } else {
    throw createI18nError(I18nErrorCodes.INVALID_VALUE);
  }
}
function makeParams(value) {
  const { path, locale, args, choice, plural } = value;
  const options = {};
  const named = args || {};
  if (isString$1(locale)) {
    options.locale = locale;
  }
  if (isNumber(choice)) {
    options.plural = choice;
  }
  if (isNumber(plural)) {
    options.plural = plural;
  }
  return [path, named, options];
}
function apply(app, i18n2, ...options) {
  const pluginOptions = isPlainObject(options[0]) ? options[0] : {};
  const useI18nComponentName = !!pluginOptions.useI18nComponentName;
  const globalInstall = isBoolean(pluginOptions.globalInstall) ? pluginOptions.globalInstall : true;
  if (globalInstall) {
    [!useI18nComponentName ? Translation.name : "i18n", "I18nT"].forEach((name) => app.component(name, Translation));
    [NumberFormat.name, "I18nN"].forEach((name) => app.component(name, NumberFormat));
    [DatetimeFormat.name, "I18nD"].forEach((name) => app.component(name, DatetimeFormat));
  }
  {
    app.directive("t", vTDirective(i18n2));
  }
}
function defineMixin(vuei18n, composer, i18n2) {
  return {
    beforeCreate() {
      const instance = getCurrentInstance();
      if (!instance) {
        throw createI18nError(I18nErrorCodes.UNEXPECTED_ERROR);
      }
      const options = this.$options;
      if (options.i18n) {
        const optionsI18n = options.i18n;
        if (options.__i18n) {
          optionsI18n.__i18n = options.__i18n;
        }
        optionsI18n.__root = composer;
        if (this === this.$root) {
          this.$i18n = mergeToGlobal(vuei18n, optionsI18n);
        } else {
          optionsI18n.__injectWithOption = true;
          optionsI18n.__extender = i18n2.__vueI18nExtend;
          this.$i18n = createVueI18n(optionsI18n);
          const _vueI18n = this.$i18n;
          if (_vueI18n.__extender) {
            _vueI18n.__disposer = _vueI18n.__extender(this.$i18n);
          }
        }
      } else if (options.__i18n) {
        if (this === this.$root) {
          this.$i18n = mergeToGlobal(vuei18n, options);
        } else {
          this.$i18n = createVueI18n({
            __i18n: options.__i18n,
            __injectWithOption: true,
            __extender: i18n2.__vueI18nExtend,
            __root: composer
          });
          const _vueI18n = this.$i18n;
          if (_vueI18n.__extender) {
            _vueI18n.__disposer = _vueI18n.__extender(this.$i18n);
          }
        }
      } else {
        this.$i18n = vuei18n;
      }
      if (options.__i18nGlobal) {
        adjustI18nResources(composer, options, options);
      }
      this.$t = (...args) => this.$i18n.t(...args);
      this.$rt = (...args) => this.$i18n.rt(...args);
      this.$tc = (...args) => this.$i18n.tc(...args);
      this.$te = (key, locale) => this.$i18n.te(key, locale);
      this.$d = (...args) => this.$i18n.d(...args);
      this.$n = (...args) => this.$i18n.n(...args);
      this.$tm = (key) => this.$i18n.tm(key);
      i18n2.__setInstance(instance, this.$i18n);
    },
    mounted() {
    },
    unmounted() {
      const instance = getCurrentInstance();
      if (!instance) {
        throw createI18nError(I18nErrorCodes.UNEXPECTED_ERROR);
      }
      const _vueI18n = this.$i18n;
      delete this.$t;
      delete this.$rt;
      delete this.$tc;
      delete this.$te;
      delete this.$d;
      delete this.$n;
      delete this.$tm;
      if (_vueI18n.__disposer) {
        _vueI18n.__disposer();
        delete _vueI18n.__disposer;
        delete _vueI18n.__extender;
      }
      i18n2.__deleteInstance(instance);
      delete this.$i18n;
    }
  };
}
function mergeToGlobal(g, options) {
  g.locale = options.locale || g.locale;
  g.fallbackLocale = options.fallbackLocale || g.fallbackLocale;
  g.missing = options.missing || g.missing;
  g.silentTranslationWarn = options.silentTranslationWarn || g.silentFallbackWarn;
  g.silentFallbackWarn = options.silentFallbackWarn || g.silentFallbackWarn;
  g.formatFallbackMessages = options.formatFallbackMessages || g.formatFallbackMessages;
  g.postTranslation = options.postTranslation || g.postTranslation;
  g.warnHtmlInMessage = options.warnHtmlInMessage || g.warnHtmlInMessage;
  g.escapeParameterHtml = options.escapeParameterHtml || g.escapeParameterHtml;
  g.sync = options.sync || g.sync;
  g.__composer[SetPluralRulesSymbol](options.pluralizationRules || g.pluralizationRules);
  const messages = getLocaleMessages(g.locale, {
    messages: options.messages,
    __i18n: options.__i18n
  });
  Object.keys(messages).forEach((locale) => g.mergeLocaleMessage(locale, messages[locale]));
  if (options.datetimeFormats) {
    Object.keys(options.datetimeFormats).forEach((locale) => g.mergeDateTimeFormat(locale, options.datetimeFormats[locale]));
  }
  if (options.numberFormats) {
    Object.keys(options.numberFormats).forEach((locale) => g.mergeNumberFormat(locale, options.numberFormats[locale]));
  }
  return g;
}
const I18nInjectionKey = /* @__PURE__ */ makeSymbol("global-vue-i18n");
function createI18n(options = {}, VueI18nLegacy) {
  const __legacyMode = __VUE_I18N_LEGACY_API__ && isBoolean(options.legacy) ? options.legacy : __VUE_I18N_LEGACY_API__;
  const __globalInjection = isBoolean(options.globalInjection) ? options.globalInjection : true;
  const __allowComposition = __VUE_I18N_LEGACY_API__ && __legacyMode ? !!options.allowComposition : true;
  const __instances = /* @__PURE__ */ new Map();
  const [globalScope, __global] = createGlobal(options, __legacyMode);
  const symbol = /* @__PURE__ */ makeSymbol("");
  function __getInstance(component) {
    return __instances.get(component) || null;
  }
  function __setInstance(component, instance) {
    __instances.set(component, instance);
  }
  function __deleteInstance(component) {
    __instances.delete(component);
  }
  {
    const i18n2 = {
      // mode
      get mode() {
        return __VUE_I18N_LEGACY_API__ && __legacyMode ? "legacy" : "composition";
      },
      // allowComposition
      get allowComposition() {
        return __allowComposition;
      },
      // install plugin
      async install(app, ...options2) {
        app.__VUE_I18N_SYMBOL__ = symbol;
        app.provide(app.__VUE_I18N_SYMBOL__, i18n2);
        if (isPlainObject(options2[0])) {
          const opts = options2[0];
          i18n2.__composerExtend = opts.__composerExtend;
          i18n2.__vueI18nExtend = opts.__vueI18nExtend;
        }
        let globalReleaseHandler = null;
        if (!__legacyMode && __globalInjection) {
          globalReleaseHandler = injectGlobalFields(app, i18n2.global);
        }
        if (__VUE_I18N_FULL_INSTALL__) {
          apply(app, i18n2, ...options2);
        }
        if (__VUE_I18N_LEGACY_API__ && __legacyMode) {
          app.mixin(defineMixin(__global, __global.__composer, i18n2));
        }
        const unmountApp = app.unmount;
        app.unmount = () => {
          globalReleaseHandler && globalReleaseHandler();
          i18n2.dispose();
          unmountApp();
        };
      },
      // global accessor
      get global() {
        return __global;
      },
      dispose() {
        globalScope.stop();
      },
      // @internal
      __instances,
      // @internal
      __getInstance,
      // @internal
      __setInstance,
      // @internal
      __deleteInstance
    };
    return i18n2;
  }
}
function useI18n(options = {}) {
  const instance = getCurrentInstance();
  if (instance == null) {
    throw createI18nError(I18nErrorCodes.MUST_BE_CALL_SETUP_TOP);
  }
  if (!instance.isCE && instance.appContext.app != null && !instance.appContext.app.__VUE_I18N_SYMBOL__) {
    throw createI18nError(I18nErrorCodes.NOT_INSTALLED);
  }
  const i18n2 = getI18nInstance(instance);
  const gl = getGlobalComposer(i18n2);
  const componentOptions = getComponentOptions(instance);
  const scope = getScope(options, componentOptions);
  if (__VUE_I18N_LEGACY_API__) {
    if (i18n2.mode === "legacy" && !options.__useComponent) {
      if (!i18n2.allowComposition) {
        throw createI18nError(I18nErrorCodes.NOT_AVAILABLE_IN_LEGACY_MODE);
      }
      return useI18nForLegacy(instance, scope, gl, options);
    }
  }
  if (scope === "global") {
    adjustI18nResources(gl, options, componentOptions);
    return gl;
  }
  if (scope === "parent") {
    let composer2 = getComposer(i18n2, instance, options.__useComponent);
    if (composer2 == null) {
      composer2 = gl;
    }
    return composer2;
  }
  const i18nInternal = i18n2;
  let composer = i18nInternal.__getInstance(instance);
  if (composer == null) {
    const composerOptions = assign$1({}, options);
    if ("__i18n" in componentOptions) {
      composerOptions.__i18n = componentOptions.__i18n;
    }
    if (gl) {
      composerOptions.__root = gl;
    }
    composer = createComposer(composerOptions);
    if (i18nInternal.__composerExtend) {
      composer[DisposeSymbol] = i18nInternal.__composerExtend(composer);
    }
    setupLifeCycle(i18nInternal, instance, composer);
    i18nInternal.__setInstance(instance, composer);
  }
  return composer;
}
function createGlobal(options, legacyMode, VueI18nLegacy) {
  const scope = effectScope();
  {
    const obj = __VUE_I18N_LEGACY_API__ && legacyMode ? scope.run(() => createVueI18n(options)) : scope.run(() => createComposer(options));
    if (obj == null) {
      throw createI18nError(I18nErrorCodes.UNEXPECTED_ERROR);
    }
    return [scope, obj];
  }
}
function getI18nInstance(instance) {
  {
    const i18n2 = inject(!instance.isCE ? instance.appContext.app.__VUE_I18N_SYMBOL__ : I18nInjectionKey);
    if (!i18n2) {
      throw createI18nError(!instance.isCE ? I18nErrorCodes.UNEXPECTED_ERROR : I18nErrorCodes.NOT_INSTALLED_WITH_PROVIDE);
    }
    return i18n2;
  }
}
function getScope(options, componentOptions) {
  return isEmptyObject(options) ? "__i18n" in componentOptions ? "local" : "global" : !options.useScope ? "local" : options.useScope;
}
function getGlobalComposer(i18n2) {
  return i18n2.mode === "composition" ? i18n2.global : i18n2.global.__composer;
}
function getComposer(i18n2, target, useComponent = false) {
  let composer = null;
  const root = target.root;
  let current = getParentComponentInstance(target, useComponent);
  while (current != null) {
    const i18nInternal = i18n2;
    if (i18n2.mode === "composition") {
      composer = i18nInternal.__getInstance(current);
    } else {
      if (__VUE_I18N_LEGACY_API__) {
        const vueI18n = i18nInternal.__getInstance(current);
        if (vueI18n != null) {
          composer = vueI18n.__composer;
          if (useComponent && composer && !composer[InejctWithOptionSymbol]) {
            composer = null;
          }
        }
      }
    }
    if (composer != null) {
      break;
    }
    if (root === current) {
      break;
    }
    current = current.parent;
  }
  return composer;
}
function getParentComponentInstance(target, useComponent = false) {
  if (target == null) {
    return null;
  }
  {
    return !useComponent ? target.parent : target.vnode.ctx || target.parent;
  }
}
function setupLifeCycle(i18n2, target, composer) {
  {
    onMounted(() => {
    }, target);
    onUnmounted(() => {
      const _composer = composer;
      i18n2.__deleteInstance(target);
      const dispose = _composer[DisposeSymbol];
      if (dispose) {
        dispose();
        delete _composer[DisposeSymbol];
      }
    }, target);
  }
}
function useI18nForLegacy(instance, scope, root, options = {}) {
  const isLocalScope = scope === "local";
  const _composer = /* @__PURE__ */ shallowRef(null);
  if (isLocalScope && instance.proxy && !(instance.proxy.$options.i18n || instance.proxy.$options.__i18n)) {
    throw createI18nError(I18nErrorCodes.MUST_DEFINE_I18N_OPTION_IN_ALLOW_COMPOSITION);
  }
  const _inheritLocale = isBoolean(options.inheritLocale) ? options.inheritLocale : !isString$1(options.locale);
  const _locale = /* @__PURE__ */ ref(
    // prettier-ignore
    !isLocalScope || _inheritLocale ? root.locale.value : isString$1(options.locale) ? options.locale : DEFAULT_LOCALE
  );
  const _fallbackLocale = /* @__PURE__ */ ref(
    // prettier-ignore
    !isLocalScope || _inheritLocale ? root.fallbackLocale.value : isString$1(options.fallbackLocale) || isArray(options.fallbackLocale) || isPlainObject(options.fallbackLocale) || options.fallbackLocale === false ? options.fallbackLocale : _locale.value
  );
  const _messages = /* @__PURE__ */ ref(getLocaleMessages(_locale.value, options));
  const _datetimeFormats = /* @__PURE__ */ ref(isPlainObject(options.datetimeFormats) ? options.datetimeFormats : { [_locale.value]: {} });
  const _numberFormats = /* @__PURE__ */ ref(isPlainObject(options.numberFormats) ? options.numberFormats : { [_locale.value]: {} });
  const _missingWarn = isLocalScope ? root.missingWarn : isBoolean(options.missingWarn) || isRegExp(options.missingWarn) ? options.missingWarn : true;
  const _fallbackWarn = isLocalScope ? root.fallbackWarn : isBoolean(options.fallbackWarn) || isRegExp(options.fallbackWarn) ? options.fallbackWarn : true;
  const _fallbackRoot = isLocalScope ? root.fallbackRoot : isBoolean(options.fallbackRoot) ? options.fallbackRoot : true;
  const _fallbackFormat = !!options.fallbackFormat;
  const _missing = isFunction(options.missing) ? options.missing : null;
  const _postTranslation = isFunction(options.postTranslation) ? options.postTranslation : null;
  const _warnHtmlMessage = isLocalScope ? root.warnHtmlMessage : isBoolean(options.warnHtmlMessage) ? options.warnHtmlMessage : true;
  const _escapeParameter = !!options.escapeParameter;
  const _modifiers = isLocalScope ? root.modifiers : isPlainObject(options.modifiers) ? options.modifiers : {};
  const _pluralRules = options.pluralRules || isLocalScope && root.pluralRules;
  function trackReactivityValues() {
    return [
      _locale.value,
      _fallbackLocale.value,
      _messages.value,
      _datetimeFormats.value,
      _numberFormats.value
    ];
  }
  const locale = computed({
    get: () => {
      return _composer.value ? _composer.value.locale.value : _locale.value;
    },
    set: (val) => {
      if (_composer.value) {
        _composer.value.locale.value = val;
      }
      _locale.value = val;
    }
  });
  const fallbackLocale = computed({
    get: () => {
      return _composer.value ? _composer.value.fallbackLocale.value : _fallbackLocale.value;
    },
    set: (val) => {
      if (_composer.value) {
        _composer.value.fallbackLocale.value = val;
      }
      _fallbackLocale.value = val;
    }
  });
  const messages = computed(() => {
    if (_composer.value) {
      return _composer.value.messages.value;
    } else {
      return _messages.value;
    }
  });
  const datetimeFormats = computed(() => _datetimeFormats.value);
  const numberFormats = computed(() => _numberFormats.value);
  function getPostTranslationHandler() {
    return _composer.value ? _composer.value.getPostTranslationHandler() : _postTranslation;
  }
  function setPostTranslationHandler(handler) {
    if (_composer.value) {
      _composer.value.setPostTranslationHandler(handler);
    }
  }
  function getMissingHandler() {
    return _composer.value ? _composer.value.getMissingHandler() : _missing;
  }
  function setMissingHandler(handler) {
    if (_composer.value) {
      _composer.value.setMissingHandler(handler);
    }
  }
  function warpWithDeps(fn) {
    trackReactivityValues();
    return fn();
  }
  function t(...args) {
    return _composer.value ? warpWithDeps(() => Reflect.apply(_composer.value.t, null, [...args])) : warpWithDeps(() => "");
  }
  function rt(...args) {
    return _composer.value ? Reflect.apply(_composer.value.rt, null, [...args]) : "";
  }
  function d(...args) {
    return _composer.value ? warpWithDeps(() => Reflect.apply(_composer.value.d, null, [...args])) : warpWithDeps(() => "");
  }
  function n(...args) {
    return _composer.value ? warpWithDeps(() => Reflect.apply(_composer.value.n, null, [...args])) : warpWithDeps(() => "");
  }
  function tm(key) {
    return _composer.value ? _composer.value.tm(key) : {};
  }
  function te(key, locale2) {
    return _composer.value ? _composer.value.te(key, locale2) : false;
  }
  function getLocaleMessage(locale2) {
    return _composer.value ? _composer.value.getLocaleMessage(locale2) : {};
  }
  function setLocaleMessage(locale2, message) {
    if (_composer.value) {
      _composer.value.setLocaleMessage(locale2, message);
      _messages.value[locale2] = message;
    }
  }
  function mergeLocaleMessage(locale2, message) {
    if (_composer.value) {
      _composer.value.mergeLocaleMessage(locale2, message);
    }
  }
  function getDateTimeFormat(locale2) {
    return _composer.value ? _composer.value.getDateTimeFormat(locale2) : {};
  }
  function setDateTimeFormat(locale2, format2) {
    if (_composer.value) {
      _composer.value.setDateTimeFormat(locale2, format2);
      _datetimeFormats.value[locale2] = format2;
    }
  }
  function mergeDateTimeFormat(locale2, format2) {
    if (_composer.value) {
      _composer.value.mergeDateTimeFormat(locale2, format2);
    }
  }
  function getNumberFormat(locale2) {
    return _composer.value ? _composer.value.getNumberFormat(locale2) : {};
  }
  function setNumberFormat(locale2, format2) {
    if (_composer.value) {
      _composer.value.setNumberFormat(locale2, format2);
      _numberFormats.value[locale2] = format2;
    }
  }
  function mergeNumberFormat(locale2, format2) {
    if (_composer.value) {
      _composer.value.mergeNumberFormat(locale2, format2);
    }
  }
  const wrapper = {
    get id() {
      return _composer.value ? _composer.value.id : -1;
    },
    locale,
    fallbackLocale,
    messages,
    datetimeFormats,
    numberFormats,
    get inheritLocale() {
      return _composer.value ? _composer.value.inheritLocale : _inheritLocale;
    },
    set inheritLocale(val) {
      if (_composer.value) {
        _composer.value.inheritLocale = val;
      }
    },
    get availableLocales() {
      return _composer.value ? _composer.value.availableLocales : Object.keys(_messages.value);
    },
    get modifiers() {
      return _composer.value ? _composer.value.modifiers : _modifiers;
    },
    get pluralRules() {
      return _composer.value ? _composer.value.pluralRules : _pluralRules;
    },
    get isGlobal() {
      return _composer.value ? _composer.value.isGlobal : false;
    },
    get missingWarn() {
      return _composer.value ? _composer.value.missingWarn : _missingWarn;
    },
    set missingWarn(val) {
      if (_composer.value) {
        _composer.value.missingWarn = val;
      }
    },
    get fallbackWarn() {
      return _composer.value ? _composer.value.fallbackWarn : _fallbackWarn;
    },
    set fallbackWarn(val) {
      if (_composer.value) {
        _composer.value.missingWarn = val;
      }
    },
    get fallbackRoot() {
      return _composer.value ? _composer.value.fallbackRoot : _fallbackRoot;
    },
    set fallbackRoot(val) {
      if (_composer.value) {
        _composer.value.fallbackRoot = val;
      }
    },
    get fallbackFormat() {
      return _composer.value ? _composer.value.fallbackFormat : _fallbackFormat;
    },
    set fallbackFormat(val) {
      if (_composer.value) {
        _composer.value.fallbackFormat = val;
      }
    },
    get warnHtmlMessage() {
      return _composer.value ? _composer.value.warnHtmlMessage : _warnHtmlMessage;
    },
    set warnHtmlMessage(val) {
      if (_composer.value) {
        _composer.value.warnHtmlMessage = val;
      }
    },
    get escapeParameter() {
      return _composer.value ? _composer.value.escapeParameter : _escapeParameter;
    },
    set escapeParameter(val) {
      if (_composer.value) {
        _composer.value.escapeParameter = val;
      }
    },
    t,
    getPostTranslationHandler,
    setPostTranslationHandler,
    getMissingHandler,
    setMissingHandler,
    rt,
    d,
    n,
    tm,
    te,
    getLocaleMessage,
    setLocaleMessage,
    mergeLocaleMessage,
    getDateTimeFormat,
    setDateTimeFormat,
    mergeDateTimeFormat,
    getNumberFormat,
    setNumberFormat,
    mergeNumberFormat
  };
  function sync(composer) {
    composer.locale.value = _locale.value;
    composer.fallbackLocale.value = _fallbackLocale.value;
    Object.keys(_messages.value).forEach((locale2) => {
      composer.mergeLocaleMessage(locale2, _messages.value[locale2]);
    });
    Object.keys(_datetimeFormats.value).forEach((locale2) => {
      composer.mergeDateTimeFormat(locale2, _datetimeFormats.value[locale2]);
    });
    Object.keys(_numberFormats.value).forEach((locale2) => {
      composer.mergeNumberFormat(locale2, _numberFormats.value[locale2]);
    });
    composer.escapeParameter = _escapeParameter;
    composer.fallbackFormat = _fallbackFormat;
    composer.fallbackRoot = _fallbackRoot;
    composer.fallbackWarn = _fallbackWarn;
    composer.missingWarn = _missingWarn;
    composer.warnHtmlMessage = _warnHtmlMessage;
  }
  onBeforeMount(() => {
    if (instance.proxy == null || instance.proxy.$i18n == null) {
      throw createI18nError(I18nErrorCodes.NOT_AVAILABLE_COMPOSITION_IN_LEGACY);
    }
    const composer = _composer.value = instance.proxy.$i18n.__composer;
    if (scope === "global") {
      _locale.value = composer.locale.value;
      _fallbackLocale.value = composer.fallbackLocale.value;
      _messages.value = composer.messages.value;
      _datetimeFormats.value = composer.datetimeFormats.value;
      _numberFormats.value = composer.numberFormats.value;
    } else if (isLocalScope) {
      sync(composer);
    }
  });
  return wrapper;
}
const globalExportProps = [
  "locale",
  "fallbackLocale",
  "availableLocales"
];
const globalExportMethods = ["t", "rt", "d", "n", "tm", "te"];
function injectGlobalFields(app, composer) {
  const i18n2 = /* @__PURE__ */ Object.create(null);
  globalExportProps.forEach((prop) => {
    const desc = Object.getOwnPropertyDescriptor(composer, prop);
    if (!desc) {
      throw createI18nError(I18nErrorCodes.UNEXPECTED_ERROR);
    }
    const wrap = /* @__PURE__ */ isRef(desc.value) ? {
      get() {
        return desc.value.value;
      },
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      set(val) {
        desc.value.value = val;
      }
    } : {
      get() {
        return desc.get && desc.get();
      }
    };
    Object.defineProperty(i18n2, prop, wrap);
  });
  app.config.globalProperties.$i18n = i18n2;
  globalExportMethods.forEach((method) => {
    const desc = Object.getOwnPropertyDescriptor(composer, method);
    if (!desc || !desc.value) {
      throw createI18nError(I18nErrorCodes.UNEXPECTED_ERROR);
    }
    Object.defineProperty(app.config.globalProperties, `$${method}`, desc);
  });
  const dispose = () => {
    delete app.config.globalProperties.$i18n;
    globalExportMethods.forEach((method) => {
      delete app.config.globalProperties[`$${method}`];
    });
  };
  return dispose;
}
{
  initFeatureFlags();
}
if (__INTLIFY_JIT_COMPILATION__) {
  registerMessageCompiler(compile);
} else {
  registerMessageCompiler(compileToFunction);
}
registerMessageResolver(resolveValue);
registerLocaleFallbacker(fallbackWithLocaleChain);
if (__INTLIFY_PROD_DEVTOOLS__) {
  const target = getGlobalThis();
  target.__INTLIFY__ = true;
  setDevToolsHook(target.__INTLIFY_DEVTOOLS_GLOBAL_HOOK__);
}
const matchIconName = /^[a-z0-9]+(-[a-z0-9]+)*$/;
const stringToIcon = (value, validate, allowSimpleName, provider = "") => {
  const colonSeparated = value.split(":");
  if (value.slice(0, 1) === "@") {
    if (colonSeparated.length < 2 || colonSeparated.length > 3) return null;
    provider = colonSeparated.shift().slice(1);
  }
  if (colonSeparated.length > 3 || !colonSeparated.length) return null;
  if (colonSeparated.length > 1) {
    const name2 = colonSeparated.pop();
    const prefix = colonSeparated.pop();
    const result = {
      provider: colonSeparated.length > 0 ? colonSeparated[0] : provider,
      prefix,
      name: name2
    };
    return validate && !validateIconName(result) ? null : result;
  }
  const name = colonSeparated[0];
  const dashSeparated = name.split("-");
  if (dashSeparated.length > 1) {
    const result = {
      provider,
      prefix: dashSeparated.shift(),
      name: dashSeparated.join("-")
    };
    return validate && !validateIconName(result) ? null : result;
  }
  if (allowSimpleName && provider === "") {
    const result = {
      provider,
      prefix: "",
      name
    };
    return validate && !validateIconName(result, allowSimpleName) ? null : result;
  }
  return null;
};
const validateIconName = (icon, allowSimpleName) => {
  if (!icon) return false;
  return !!((allowSimpleName && icon.prefix === "" || !!icon.prefix) && !!icon.name);
};
function getIconsTree(data, names) {
  const icons = data.icons;
  const aliases = data.aliases || /* @__PURE__ */ Object.create(null);
  const resolved = /* @__PURE__ */ Object.create(null);
  function resolve2(name) {
    if (icons[name]) return resolved[name] = [];
    if (!(name in resolved)) {
      resolved[name] = null;
      const parent = aliases[name] && aliases[name].parent;
      const value = parent && resolve2(parent);
      if (value) resolved[name] = [parent].concat(value);
    }
    return resolved[name];
  }
  Object.keys(icons).concat(Object.keys(aliases)).forEach(resolve2);
  return resolved;
}
const defaultIconDimensions = Object.freeze({
  left: 0,
  top: 0,
  width: 16,
  height: 16
});
const defaultIconTransformations = Object.freeze({
  rotate: 0,
  vFlip: false,
  hFlip: false
});
const defaultIconProps = Object.freeze({
  ...defaultIconDimensions,
  ...defaultIconTransformations
});
const defaultExtendedIconProps = Object.freeze({
  ...defaultIconProps,
  body: "",
  hidden: false
});
function mergeIconTransformations(obj1, obj2) {
  const result = {};
  if (!obj1.hFlip !== !obj2.hFlip) result.hFlip = true;
  if (!obj1.vFlip !== !obj2.vFlip) result.vFlip = true;
  const rotate = ((obj1.rotate || 0) + (obj2.rotate || 0)) % 4;
  if (rotate) result.rotate = rotate;
  return result;
}
function mergeIconData(parent, child) {
  const result = mergeIconTransformations(parent, child);
  for (const key in defaultExtendedIconProps) if (key in defaultIconTransformations) {
    if (key in parent && !(key in result)) result[key] = defaultIconTransformations[key];
  } else if (key in child) result[key] = child[key];
  else if (key in parent) result[key] = parent[key];
  return result;
}
function internalGetIconData(data, name, tree) {
  const icons = data.icons;
  const aliases = data.aliases || /* @__PURE__ */ Object.create(null);
  let currentProps = {};
  function parse2(name2) {
    currentProps = mergeIconData(icons[name2] || aliases[name2], currentProps);
  }
  parse2(name);
  tree.forEach(parse2);
  return mergeIconData(data, currentProps);
}
function parseIconSet(data, callback) {
  const names = [];
  if (typeof data !== "object" || typeof data.icons !== "object") return names;
  if (data.not_found instanceof Array) data.not_found.forEach((name) => {
    callback(name, null);
    names.push(name);
  });
  const tree = getIconsTree(data);
  for (const name in tree) {
    const item = tree[name];
    if (item) {
      callback(name, internalGetIconData(data, name, item));
      names.push(name);
    }
  }
  return names;
}
const optionalPropertyDefaults = {
  provider: "",
  aliases: {},
  not_found: {},
  ...defaultIconDimensions
};
function checkOptionalProps(item, defaults) {
  for (const prop in defaults) if (prop in item && typeof item[prop] !== typeof defaults[prop]) return false;
  return true;
}
function quicklyValidateIconSet(obj) {
  if (typeof obj !== "object" || obj === null) return null;
  const data = obj;
  if (typeof data.prefix !== "string" || !obj.icons || typeof obj.icons !== "object") return null;
  if (!checkOptionalProps(obj, optionalPropertyDefaults)) return null;
  const icons = data.icons;
  for (const name in icons) {
    const icon = icons[name];
    if (!name || typeof icon.body !== "string" || !checkOptionalProps(icon, defaultExtendedIconProps)) return null;
  }
  const aliases = data.aliases || /* @__PURE__ */ Object.create(null);
  for (const name in aliases) {
    const icon = aliases[name];
    const parent = icon.parent;
    if (!name || typeof parent !== "string" || !icons[parent] && !aliases[parent] || !checkOptionalProps(icon, defaultExtendedIconProps)) return null;
  }
  return data;
}
const dataStorage = /* @__PURE__ */ Object.create(null);
function newStorage(provider, prefix) {
  return {
    provider,
    prefix,
    icons: /* @__PURE__ */ Object.create(null),
    missing: /* @__PURE__ */ new Set()
  };
}
function getStorage(provider, prefix) {
  const providerStorage = dataStorage[provider] || (dataStorage[provider] = /* @__PURE__ */ Object.create(null));
  return providerStorage[prefix] || (providerStorage[prefix] = newStorage(provider, prefix));
}
function addIconSet(storage2, data) {
  if (!quicklyValidateIconSet(data)) return [];
  return parseIconSet(data, (name, icon) => {
    if (icon) storage2.icons[name] = icon;
    else storage2.missing.add(name);
  });
}
function addIconToStorage(storage2, name, icon) {
  try {
    if (typeof icon.body === "string") {
      storage2.icons[name] = { ...icon };
      return true;
    }
  } catch (err) {
  }
  return false;
}
let simpleNames = false;
function allowSimpleNames(allow) {
  if (typeof allow === "boolean") simpleNames = allow;
  return simpleNames;
}
function getIconData(name) {
  const icon = typeof name === "string" ? stringToIcon(name, true, simpleNames) : name;
  if (icon) {
    const storage2 = getStorage(icon.provider, icon.prefix);
    const iconName = icon.name;
    return storage2.icons[iconName] || (storage2.missing.has(iconName) ? null : void 0);
  }
}
function addIcon(name, data) {
  const icon = stringToIcon(name, true, simpleNames);
  if (!icon) return false;
  const storage2 = getStorage(icon.provider, icon.prefix);
  if (data) return addIconToStorage(storage2, icon.name, data);
  else {
    storage2.missing.add(icon.name);
    return true;
  }
}
function addCollection(data, provider) {
  if (typeof data !== "object") return false;
  if (typeof provider !== "string") provider = data.provider || "";
  if (simpleNames && !provider && !data.prefix) {
    let added = false;
    if (quicklyValidateIconSet(data)) {
      data.prefix = "";
      parseIconSet(data, (name, icon) => {
        if (addIcon(name, icon)) added = true;
      });
    }
    return added;
  }
  const prefix = data.prefix;
  if (!validateIconName({
    prefix,
    name: "a"
  })) return false;
  return !!addIconSet(getStorage(provider, prefix), data);
}
const defaultIconSizeCustomisations = Object.freeze({
  width: null,
  height: null
});
const defaultIconCustomisations = Object.freeze({
  ...defaultIconSizeCustomisations,
  ...defaultIconTransformations
});
const unitsSplit = /(-?[0-9.]*[0-9]+[0-9.]*)/g;
const unitsTest = /^-?[0-9.]*[0-9]+[0-9.]*$/g;
function calculateSize(size2, ratio, precision) {
  if (ratio === 1) return size2;
  precision = precision || 100;
  if (typeof size2 === "number") return Math.ceil(size2 * ratio * precision) / precision;
  if (typeof size2 !== "string") return size2;
  const oldParts = size2.split(unitsSplit);
  if (oldParts === null || !oldParts.length) return size2;
  const newParts = [];
  let code2 = oldParts.shift();
  let isNumber2 = unitsTest.test(code2);
  while (true) {
    if (isNumber2) {
      const num = parseFloat(code2);
      if (isNaN(num)) newParts.push(code2);
      else newParts.push(Math.ceil(num * ratio * precision) / precision);
    } else newParts.push(code2);
    code2 = oldParts.shift();
    if (code2 === void 0) return newParts.join("");
    isNumber2 = !isNumber2;
  }
}
function splitSVGDefs(content, tag = "defs") {
  let defs = "";
  const index = content.indexOf("<" + tag);
  while (index >= 0) {
    const start = content.indexOf(">", index);
    const end = content.indexOf("</" + tag);
    if (start === -1 || end === -1) break;
    const endEnd = content.indexOf(">", end);
    if (endEnd === -1) break;
    defs += content.slice(start + 1, end).trim();
    content = content.slice(0, index).trim() + content.slice(endEnd + 1);
  }
  return {
    defs,
    content
  };
}
function mergeDefsAndContent(defs, content) {
  return defs ? "<defs>" + defs + "</defs>" + content : content;
}
function wrapSVGContent(body, start, end) {
  const split = splitSVGDefs(body);
  return mergeDefsAndContent(split.defs, start + split.content + end);
}
const isUnsetKeyword = (value) => value === "unset" || value === "undefined" || value === "none";
function iconToSVG(icon, customisations) {
  const fullIcon = {
    ...defaultIconProps,
    ...icon
  };
  const fullCustomisations = {
    ...defaultIconCustomisations,
    ...customisations
  };
  const box = {
    left: fullIcon.left,
    top: fullIcon.top,
    width: fullIcon.width,
    height: fullIcon.height
  };
  let body = fullIcon.body;
  [fullIcon, fullCustomisations].forEach((props) => {
    const transformations = [];
    const hFlip = props.hFlip;
    const vFlip = props.vFlip;
    let rotation = props.rotate;
    if (hFlip) if (vFlip) rotation += 2;
    else {
      transformations.push("translate(" + (box.width + box.left).toString() + " " + (0 - box.top).toString() + ")");
      transformations.push("scale(-1 1)");
      box.top = box.left = 0;
    }
    else if (vFlip) {
      transformations.push("translate(" + (0 - box.left).toString() + " " + (box.height + box.top).toString() + ")");
      transformations.push("scale(1 -1)");
      box.top = box.left = 0;
    }
    let tempValue;
    if (rotation < 0) rotation -= Math.floor(rotation / 4) * 4;
    rotation = rotation % 4;
    switch (rotation) {
      case 1:
        tempValue = box.height / 2 + box.top;
        transformations.unshift("rotate(90 " + tempValue.toString() + " " + tempValue.toString() + ")");
        break;
      case 2:
        transformations.unshift("rotate(180 " + (box.width / 2 + box.left).toString() + " " + (box.height / 2 + box.top).toString() + ")");
        break;
      case 3:
        tempValue = box.width / 2 + box.left;
        transformations.unshift("rotate(-90 " + tempValue.toString() + " " + tempValue.toString() + ")");
        break;
    }
    if (rotation % 2 === 1) {
      if (box.left !== box.top) {
        tempValue = box.left;
        box.left = box.top;
        box.top = tempValue;
      }
      if (box.width !== box.height) {
        tempValue = box.width;
        box.width = box.height;
        box.height = tempValue;
      }
    }
    if (transformations.length) body = wrapSVGContent(body, '<g transform="' + transformations.join(" ") + '">', "</g>");
  });
  const customisationsWidth = fullCustomisations.width;
  const customisationsHeight = fullCustomisations.height;
  const boxWidth = box.width;
  const boxHeight = box.height;
  let width;
  let height;
  if (customisationsWidth === null) {
    height = customisationsHeight === null ? "1em" : customisationsHeight === "auto" ? boxHeight : customisationsHeight;
    width = calculateSize(height, boxWidth / boxHeight);
  } else {
    width = customisationsWidth === "auto" ? boxWidth : customisationsWidth;
    height = customisationsHeight === null ? calculateSize(width, boxHeight / boxWidth) : customisationsHeight === "auto" ? boxHeight : customisationsHeight;
  }
  const attributes = {};
  const setAttr = (prop, value) => {
    if (!isUnsetKeyword(value)) attributes[prop] = value.toString();
  };
  setAttr("width", width);
  setAttr("height", height);
  const viewBox = [
    box.left,
    box.top,
    boxWidth,
    boxHeight
  ];
  attributes.viewBox = viewBox.join(" ");
  return {
    attributes,
    viewBox,
    body
  };
}
const regex = /\sid="(\S+)"/g;
const counters = /* @__PURE__ */ new Map();
function nextID(id) {
  id = id.replace(/[0-9]+$/, "") || "a";
  const count = counters.get(id) || 0;
  counters.set(id, count + 1);
  return count ? `${id}${count}` : id;
}
function replaceIDs(body) {
  const ids = [];
  let match;
  while (match = regex.exec(body)) ids.push(match[1]);
  if (!ids.length) return body;
  const suffix = "suffix" + (Math.random() * 16777216 | Date.now()).toString(16);
  ids.forEach((id) => {
    const newID = nextID(id);
    const escapedID = id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    body = body.replace(new RegExp('([#;"])(' + escapedID + ')([")]|\\.[a-z])', "g"), "$1" + newID + suffix + "$3");
  });
  body = body.replace(new RegExp(suffix, "g"), "");
  return body;
}
const storage = /* @__PURE__ */ Object.create(null);
function setAPIModule(provider, item) {
  storage[provider] = item;
}
function getAPIModule(provider) {
  return storage[provider] || storage[""];
}
function createAPIConfig(source) {
  let resources;
  if (typeof source.resources === "string") resources = [source.resources];
  else {
    resources = source.resources;
    if (!(resources instanceof Array) || !resources.length) return null;
  }
  return {
    resources,
    path: source.path || "/",
    maxURL: source.maxURL || 500,
    rotate: source.rotate || 750,
    timeout: source.timeout || 5e3,
    random: source.random === true,
    index: source.index || 0,
    dataAfterTimeout: source.dataAfterTimeout !== false
  };
}
const configStorage = /* @__PURE__ */ Object.create(null);
const fallBackAPISources = ["https://api.simplesvg.com", "https://api.unisvg.com"];
const fallBackAPI = [];
while (fallBackAPISources.length > 0) if (fallBackAPISources.length === 1) fallBackAPI.push(fallBackAPISources.shift());
else if (Math.random() > 0.5) fallBackAPI.push(fallBackAPISources.shift());
else fallBackAPI.push(fallBackAPISources.pop());
configStorage[""] = createAPIConfig({ resources: ["https://api.iconify.design"].concat(fallBackAPI) });
function addAPIProvider(provider, customConfig) {
  const config = createAPIConfig(customConfig);
  if (config === null) return false;
  configStorage[provider] = config;
  return true;
}
function getAPIConfig(provider) {
  return configStorage[provider];
}
const detectFetch = () => {
  let callback;
  try {
    callback = fetch;
    if (typeof callback === "function") return callback;
  } catch (err) {
  }
};
let fetchModule = detectFetch();
function calculateMaxLength(provider, prefix) {
  const config = getAPIConfig(provider);
  if (!config) return 0;
  let result;
  if (!config.maxURL) result = 0;
  else {
    let maxHostLength = 0;
    config.resources.forEach((item) => {
      maxHostLength = Math.max(maxHostLength, item.length);
    });
    const url = prefix + ".json?icons=";
    result = config.maxURL - maxHostLength - config.path.length - url.length;
  }
  return result;
}
function shouldAbort(status) {
  return status === 404;
}
const prepare = (provider, prefix, icons) => {
  const results = [];
  const maxLength = calculateMaxLength(provider, prefix);
  const type = "icons";
  let item = {
    type,
    provider,
    prefix,
    icons: []
  };
  let length = 0;
  icons.forEach((name, index) => {
    length += name.length + 1;
    if (length >= maxLength && index > 0) {
      results.push(item);
      item = {
        type,
        provider,
        prefix,
        icons: []
      };
      length = name.length;
    }
    item.icons.push(name);
  });
  results.push(item);
  return results;
};
function getPath(provider) {
  if (typeof provider === "string") {
    const config = getAPIConfig(provider);
    if (config) return config.path;
  }
  return "/";
}
const send = (host, params, callback) => {
  if (!fetchModule) {
    callback("abort", 424);
    return;
  }
  let path = getPath(params.provider);
  switch (params.type) {
    case "icons": {
      const prefix = params.prefix;
      const iconsList = params.icons.join(",");
      const urlParams = new URLSearchParams({ icons: iconsList });
      path += prefix + ".json?" + urlParams.toString();
      break;
    }
    case "custom": {
      const uri = params.uri;
      path += uri.slice(0, 1) === "/" ? uri.slice(1) : uri;
      break;
    }
    default:
      callback("abort", 400);
      return;
  }
  let defaultError = 503;
  fetchModule(host + path).then((response2) => {
    const status = response2.status;
    if (status !== 200) {
      setTimeout(() => {
        callback(shouldAbort(status) ? "abort" : "next", status);
      });
      return;
    }
    defaultError = 501;
    return response2.json();
  }).then((data) => {
    if (typeof data !== "object" || data === null) {
      setTimeout(() => {
        if (data === 404) callback("abort", data);
        else callback("next", defaultError);
      });
      return;
    }
    setTimeout(() => {
      callback("success", data);
    });
  }).catch(() => {
    callback("next", defaultError);
  });
};
const fetchAPIModule = {
  prepare,
  send
};
function removeCallback(storages, id) {
  storages.forEach((storage2) => {
    const items = storage2.loaderCallbacks;
    if (items) storage2.loaderCallbacks = items.filter((row) => row.id !== id);
  });
}
function updateCallbacks(storage2) {
  if (!storage2.pendingCallbacksFlag) {
    storage2.pendingCallbacksFlag = true;
    setTimeout(() => {
      storage2.pendingCallbacksFlag = false;
      const items = storage2.loaderCallbacks ? storage2.loaderCallbacks.slice(0) : [];
      if (!items.length) return;
      let hasPending = false;
      const provider = storage2.provider;
      const prefix = storage2.prefix;
      items.forEach((item) => {
        const icons = item.icons;
        const oldLength = icons.pending.length;
        icons.pending = icons.pending.filter((icon) => {
          if (icon.prefix !== prefix) return true;
          const name = icon.name;
          if (storage2.icons[name]) icons.loaded.push({
            provider,
            prefix,
            name
          });
          else if (storage2.missing.has(name)) icons.missing.push({
            provider,
            prefix,
            name
          });
          else {
            hasPending = true;
            return true;
          }
          return false;
        });
        if (icons.pending.length !== oldLength) {
          if (!hasPending) removeCallback([storage2], item.id);
          item.callback(icons.loaded.slice(0), icons.missing.slice(0), icons.pending.slice(0), item.abort);
        }
      });
    });
  }
}
let idCounter = 0;
function storeCallback(callback, icons, pendingSources) {
  const id = idCounter++;
  const abort = removeCallback.bind(null, pendingSources, id);
  if (!icons.pending.length) return abort;
  const item = {
    id,
    icons,
    callback,
    abort
  };
  pendingSources.forEach((storage2) => {
    (storage2.loaderCallbacks || (storage2.loaderCallbacks = [])).push(item);
  });
  return abort;
}
function sortIcons(icons) {
  const result = {
    loaded: [],
    missing: [],
    pending: []
  };
  const storage2 = /* @__PURE__ */ Object.create(null);
  icons.sort((a, b) => {
    if (a.provider !== b.provider) return a.provider.localeCompare(b.provider);
    if (a.prefix !== b.prefix) return a.prefix.localeCompare(b.prefix);
    return a.name.localeCompare(b.name);
  });
  let lastIcon = {
    provider: "",
    prefix: "",
    name: ""
  };
  icons.forEach((icon) => {
    if (lastIcon.name === icon.name && lastIcon.prefix === icon.prefix && lastIcon.provider === icon.provider) return;
    lastIcon = icon;
    const provider = icon.provider;
    const prefix = icon.prefix;
    const name = icon.name;
    const providerStorage = storage2[provider] || (storage2[provider] = /* @__PURE__ */ Object.create(null));
    const localStorage2 = providerStorage[prefix] || (providerStorage[prefix] = getStorage(provider, prefix));
    let list;
    if (name in localStorage2.icons) list = result.loaded;
    else if (prefix === "" || localStorage2.missing.has(name)) list = result.missing;
    else list = result.pending;
    const item = {
      provider,
      prefix,
      name
    };
    list.push(item);
  });
  return result;
}
function listToIcons(list, validate = true, simpleNames2 = false) {
  const result = [];
  list.forEach((item) => {
    const icon = typeof item === "string" ? stringToIcon(item, validate, simpleNames2) : item;
    if (icon) result.push(icon);
  });
  return result;
}
const defaultConfig = {
  resources: [],
  index: 0,
  timeout: 2e3,
  rotate: 750,
  random: false,
  dataAfterTimeout: false
};
function sendQuery(config, payload, query, done) {
  const resourcesCount = config.resources.length;
  const startIndex = config.random ? Math.floor(Math.random() * resourcesCount) : config.index;
  let resources;
  if (config.random) {
    let list = config.resources.slice(0);
    resources = [];
    while (list.length > 1) {
      const nextIndex = Math.floor(Math.random() * list.length);
      resources.push(list[nextIndex]);
      list = list.slice(0, nextIndex).concat(list.slice(nextIndex + 1));
    }
    resources = resources.concat(list);
  } else resources = config.resources.slice(startIndex).concat(config.resources.slice(0, startIndex));
  const startTime = Date.now();
  let status = "pending";
  let queriesSent = 0;
  let lastError;
  let timer = null;
  let queue2 = [];
  let doneCallbacks = [];
  if (typeof done === "function") doneCallbacks.push(done);
  function resetTimer() {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
  }
  function abort() {
    if (status === "pending") status = "aborted";
    resetTimer();
    queue2.forEach((item) => {
      if (item.status === "pending") item.status = "aborted";
    });
    queue2 = [];
  }
  function subscribe(callback, overwrite) {
    if (overwrite) doneCallbacks = [];
    if (typeof callback === "function") doneCallbacks.push(callback);
  }
  function getQueryStatus() {
    return {
      startTime,
      payload,
      status,
      queriesSent,
      queriesPending: queue2.length,
      subscribe,
      abort
    };
  }
  function failQuery() {
    status = "failed";
    doneCallbacks.forEach((callback) => {
      callback(void 0, lastError);
    });
  }
  function clearQueue() {
    queue2.forEach((item) => {
      if (item.status === "pending") item.status = "aborted";
    });
    queue2 = [];
  }
  function moduleResponse(item, response2, data) {
    const isError = response2 !== "success";
    queue2 = queue2.filter((queued) => queued !== item);
    switch (status) {
      case "pending":
        break;
      case "failed":
        if (isError || !config.dataAfterTimeout) return;
        break;
      default:
        return;
    }
    if (response2 === "abort") {
      lastError = data;
      failQuery();
      return;
    }
    if (isError) {
      lastError = data;
      if (!queue2.length) if (!resources.length) failQuery();
      else execNext();
      return;
    }
    resetTimer();
    clearQueue();
    if (!config.random) {
      const index = config.resources.indexOf(item.resource);
      if (index !== -1 && index !== config.index) config.index = index;
    }
    status = "completed";
    doneCallbacks.forEach((callback) => {
      callback(data);
    });
  }
  function execNext() {
    if (status !== "pending") return;
    resetTimer();
    const resource = resources.shift();
    if (resource === void 0) {
      if (queue2.length) {
        timer = setTimeout(() => {
          resetTimer();
          if (status === "pending") {
            clearQueue();
            failQuery();
          }
        }, config.timeout);
        return;
      }
      failQuery();
      return;
    }
    const item = {
      status: "pending",
      resource,
      callback: (status2, data) => {
        moduleResponse(item, status2, data);
      }
    };
    queue2.push(item);
    queriesSent++;
    timer = setTimeout(execNext, config.rotate);
    query(resource, payload, item.callback);
  }
  setTimeout(execNext);
  return getQueryStatus;
}
function initRedundancy(cfg) {
  const config = {
    ...defaultConfig,
    ...cfg
  };
  let queries = [];
  function cleanup() {
    queries = queries.filter((item) => item().status === "pending");
  }
  function query(payload, queryCallback, doneCallback) {
    const query2 = sendQuery(config, payload, queryCallback, (data, error) => {
      cleanup();
      if (doneCallback) doneCallback(data, error);
    });
    queries.push(query2);
    return query2;
  }
  function find(callback) {
    return queries.find((value) => {
      return callback(value);
    }) || null;
  }
  return {
    query,
    find,
    setIndex: (index) => {
      config.index = index;
    },
    getIndex: () => config.index,
    cleanup
  };
}
function emptyCallback$1() {
}
const redundancyCache = /* @__PURE__ */ Object.create(null);
function getRedundancyCache(provider) {
  if (!redundancyCache[provider]) {
    const config = getAPIConfig(provider);
    if (!config) return;
    redundancyCache[provider] = {
      config,
      redundancy: initRedundancy(config)
    };
  }
  return redundancyCache[provider];
}
function sendAPIQuery(target, query, callback) {
  let redundancy;
  let send2;
  if (typeof target === "string") {
    const api = getAPIModule(target);
    if (!api) {
      callback(void 0, 424);
      return emptyCallback$1;
    }
    send2 = api.send;
    const cached2 = getRedundancyCache(target);
    if (cached2) redundancy = cached2.redundancy;
  } else {
    const config = createAPIConfig(target);
    if (config) {
      redundancy = initRedundancy(config);
      const api = getAPIModule(target.resources ? target.resources[0] : "");
      if (api) send2 = api.send;
    }
  }
  if (!redundancy || !send2) {
    callback(void 0, 424);
    return emptyCallback$1;
  }
  return redundancy.query(query, send2, callback)().abort;
}
function emptyCallback() {
}
function loadedNewIcons(storage2) {
  if (!storage2.iconsLoaderFlag) {
    storage2.iconsLoaderFlag = true;
    setTimeout(() => {
      storage2.iconsLoaderFlag = false;
      updateCallbacks(storage2);
    });
  }
}
function checkIconNamesForAPI(icons) {
  const valid = [];
  const invalid = [];
  icons.forEach((name) => {
    (name.match(matchIconName) ? valid : invalid).push(name);
  });
  return {
    valid,
    invalid
  };
}
function parseLoaderResponse(storage2, icons, data) {
  function checkMissing() {
    const pending = storage2.pendingIcons;
    icons.forEach((name) => {
      if (pending) pending.delete(name);
      if (!storage2.icons[name]) storage2.missing.add(name);
    });
  }
  if (data && typeof data === "object") try {
    if (!addIconSet(storage2, data).length) {
      checkMissing();
      return;
    }
  } catch (err) {
    console.error(err);
  }
  checkMissing();
  loadedNewIcons(storage2);
}
function parsePossiblyAsyncResponse(response2, callback) {
  if (response2 instanceof Promise) response2.then((data) => {
    callback(data);
  }).catch(() => {
    callback(null);
  });
  else callback(response2);
}
function loadNewIcons(storage2, icons) {
  if (!storage2.iconsToLoad) storage2.iconsToLoad = icons;
  else storage2.iconsToLoad = storage2.iconsToLoad.concat(icons).sort();
  if (!storage2.iconsQueueFlag) {
    storage2.iconsQueueFlag = true;
    setTimeout(() => {
      storage2.iconsQueueFlag = false;
      const { provider, prefix } = storage2;
      const icons2 = storage2.iconsToLoad;
      delete storage2.iconsToLoad;
      if (!icons2 || !icons2.length) return;
      const customIconLoader = storage2.loadIcon;
      if (storage2.loadIcons && (icons2.length > 1 || !customIconLoader)) {
        parsePossiblyAsyncResponse(storage2.loadIcons(icons2, prefix, provider), (data) => {
          parseLoaderResponse(storage2, icons2, data);
        });
        return;
      }
      if (customIconLoader) {
        icons2.forEach((name) => {
          parsePossiblyAsyncResponse(customIconLoader(name, prefix, provider), (data) => {
            parseLoaderResponse(storage2, [name], data ? {
              prefix,
              icons: { [name]: data }
            } : null);
          });
        });
        return;
      }
      const { valid, invalid } = checkIconNamesForAPI(icons2);
      if (invalid.length) parseLoaderResponse(storage2, invalid, null);
      if (!valid.length) return;
      const api = prefix.match(matchIconName) ? getAPIModule(provider) : null;
      if (!api) {
        parseLoaderResponse(storage2, valid, null);
        return;
      }
      api.prepare(provider, prefix, valid).forEach((item) => {
        sendAPIQuery(provider, item, (data) => {
          parseLoaderResponse(storage2, item.icons, data);
        });
      });
    });
  }
}
const loadIcons = (icons, callback) => {
  const sortedIcons = sortIcons(listToIcons(icons, true, allowSimpleNames()));
  if (!sortedIcons.pending.length) {
    let callCallback = true;
    if (callback) setTimeout(() => {
      if (callCallback) callback(sortedIcons.loaded, sortedIcons.missing, sortedIcons.pending, emptyCallback);
    });
    return () => {
      callCallback = false;
    };
  }
  const newIcons = /* @__PURE__ */ Object.create(null);
  const sources = [];
  let lastProvider, lastPrefix;
  sortedIcons.pending.forEach((icon) => {
    const { provider, prefix } = icon;
    if (prefix === lastPrefix && provider === lastProvider) return;
    lastProvider = provider;
    lastPrefix = prefix;
    sources.push(getStorage(provider, prefix));
    const providerNewIcons = newIcons[provider] || (newIcons[provider] = /* @__PURE__ */ Object.create(null));
    if (!providerNewIcons[prefix]) providerNewIcons[prefix] = [];
  });
  sortedIcons.pending.forEach((icon) => {
    const { provider, prefix, name } = icon;
    const storage2 = getStorage(provider, prefix);
    const pendingQueue = storage2.pendingIcons || (storage2.pendingIcons = /* @__PURE__ */ new Set());
    if (!pendingQueue.has(name)) {
      pendingQueue.add(name);
      newIcons[provider][prefix].push(name);
    }
  });
  sources.forEach((storage2) => {
    const list = newIcons[storage2.provider][storage2.prefix];
    if (list.length) loadNewIcons(storage2, list);
  });
  return callback ? storeCallback(callback, sortedIcons, sources) : emptyCallback;
};
function mergeCustomisations(defaults, item) {
  const result = { ...defaults };
  for (const key in item) {
    const value = item[key];
    const valueType = typeof value;
    if (key in defaultIconSizeCustomisations) {
      if (value === null || value && (valueType === "string" || valueType === "number")) result[key] = value;
    } else if (valueType === typeof result[key]) result[key] = key === "rotate" ? value % 4 : value;
  }
  return result;
}
const separator = /[\s,]+/;
function flipFromString(custom, flip) {
  flip.split(separator).forEach((str) => {
    switch (str.trim()) {
      case "horizontal":
        custom.hFlip = true;
        break;
      case "vertical":
        custom.vFlip = true;
        break;
    }
  });
}
function rotateFromString(value, defaultValue = 0) {
  const units = value.replace(/^-?[0-9.]*/, "");
  function cleanup(value2) {
    while (value2 < 0) value2 += 4;
    return value2 % 4;
  }
  if (units === "") {
    const num = parseInt(value);
    return isNaN(num) ? 0 : cleanup(num);
  } else if (units !== value) {
    let split = 0;
    switch (units) {
      case "%":
        split = 25;
        break;
      case "deg":
        split = 90;
    }
    if (split) {
      let num = parseFloat(value.slice(0, value.length - units.length));
      if (isNaN(num)) return 0;
      num = num / split;
      return num % 1 === 0 ? cleanup(num) : 0;
    }
  }
  return defaultValue;
}
function iconToHTML(body, attributes) {
  let renderAttribsHTML = body.indexOf("xlink:") === -1 ? "" : ' xmlns:xlink="http://www.w3.org/1999/xlink"';
  for (const attr in attributes) renderAttribsHTML += " " + attr + '="' + attributes[attr] + '"';
  return '<svg xmlns="http://www.w3.org/2000/svg"' + renderAttribsHTML + ">" + body + "</svg>";
}
function encodeSVGforURL(svg) {
  return svg.replace(/"/g, "'").replace(/%/g, "%25").replace(/#/g, "%23").replace(/</g, "%3C").replace(/>/g, "%3E").replace(/\s+/g, " ");
}
function svgToData(svg) {
  return "data:image/svg+xml," + encodeSVGforURL(svg);
}
function svgToURL(svg) {
  return 'url("' + svgToData(svg) + '")';
}
const defaultExtendedIconCustomisations = {
  ...defaultIconCustomisations,
  inline: false
};
const svgDefaults = {
  "xmlns": "http://www.w3.org/2000/svg",
  "xmlns:xlink": "http://www.w3.org/1999/xlink",
  "aria-hidden": true,
  "role": "img"
};
const commonProps = {
  display: "inline-block"
};
const monotoneProps = {
  backgroundColor: "currentColor"
};
const coloredProps = {
  backgroundColor: "transparent"
};
const propsToAdd = {
  Image: "var(--svg)",
  Repeat: "no-repeat",
  Size: "100% 100%"
};
const propsToAddTo = {
  webkitMask: monotoneProps,
  mask: monotoneProps,
  background: coloredProps
};
for (const prefix in propsToAddTo) {
  const list = propsToAddTo[prefix];
  for (const prop in propsToAdd) {
    list[prefix + prop] = propsToAdd[prop];
  }
}
const customisationAliases = {};
["horizontal", "vertical"].forEach((prefix) => {
  const attr = prefix.slice(0, 1) + "Flip";
  customisationAliases[prefix + "-flip"] = attr;
  customisationAliases[prefix.slice(0, 1) + "-flip"] = attr;
  customisationAliases[prefix + "Flip"] = attr;
});
function fixSize(value) {
  return value + (value.match(/^[-0-9.]+$/) ? "px" : "");
}
const render = (icon, props) => {
  const customisations = mergeCustomisations(defaultExtendedIconCustomisations, props);
  const componentProps = { ...svgDefaults };
  const mode = props.mode || "svg";
  const style = {};
  const propsStyle = props.style;
  const customStyle = typeof propsStyle === "object" && !(propsStyle instanceof Array) ? propsStyle : {};
  for (let key in props) {
    const value = props[key];
    if (value === void 0) {
      continue;
    }
    switch (key) {
      case "icon":
      case "style":
      case "onLoad":
      case "mode":
      case "ssr":
      case "customise":
        break;
      case "inline":
      case "hFlip":
      case "vFlip":
        customisations[key] = value === true || value === "true" || value === 1;
        break;
      case "flip":
        if (typeof value === "string") {
          flipFromString(customisations, value);
        }
        break;
      case "color":
        style.color = value;
        break;
      case "rotate":
        if (typeof value === "string") {
          customisations[key] = rotateFromString(value);
        } else if (typeof value === "number") {
          customisations[key] = value;
        }
        break;
      case "ariaHidden":
      case "aria-hidden":
        if (value !== true && value !== "true") {
          delete componentProps["aria-hidden"];
        }
        break;
      default: {
        const alias = customisationAliases[key];
        if (alias) {
          if (value === true || value === "true" || value === 1) {
            customisations[alias] = true;
          }
        } else if (defaultExtendedIconCustomisations[key] === void 0) {
          componentProps[key] = value;
        }
      }
    }
  }
  const item = iconToSVG(icon, customisations);
  const renderAttribs = item.attributes;
  if (customisations.inline) {
    style.verticalAlign = "-0.125em";
  }
  if (mode === "svg") {
    componentProps.style = {
      ...style,
      ...customStyle
    };
    Object.assign(componentProps, renderAttribs);
    componentProps["innerHTML"] = replaceIDs(item.body);
    return h("svg", componentProps);
  }
  const { body, width, height } = icon;
  const useMask = mode === "mask" || (mode === "bg" ? false : body.indexOf("currentColor") !== -1);
  const html = iconToHTML(body, {
    ...renderAttribs,
    width: width + "",
    height: height + ""
  });
  componentProps.style = {
    ...style,
    "--svg": svgToURL(html),
    "width": fixSize(renderAttribs.width),
    "height": fixSize(renderAttribs.height),
    ...commonProps,
    ...useMask ? monotoneProps : coloredProps,
    ...customStyle
  };
  return h("span", componentProps);
};
allowSimpleNames(true);
setAPIModule("", fetchAPIModule);
if (typeof document !== "undefined" && typeof window !== "undefined") {
  const _window = window;
  if (_window.IconifyPreload !== void 0) {
    const preload = _window.IconifyPreload;
    const err = "Invalid IconifyPreload syntax.";
    if (typeof preload === "object" && preload !== null) {
      (preload instanceof Array ? preload : [preload]).forEach((item) => {
        try {
          if (
            // Check if item is an object and not null/array
            typeof item !== "object" || item === null || item instanceof Array || // Check for 'icons' and 'prefix'
            typeof item.icons !== "object" || typeof item.prefix !== "string" || // Add icon set
            !addCollection(item)
          ) {
            console.error(err);
          }
        } catch (e) {
          console.error(err);
        }
      });
    }
  }
  if (_window.IconifyProviders !== void 0) {
    const providers = _window.IconifyProviders;
    if (typeof providers === "object" && providers !== null) {
      for (let key in providers) {
        const err = "IconifyProviders[" + key + "] is invalid.";
        try {
          const value = providers[key];
          if (typeof value !== "object" || !value || value.resources === void 0) {
            continue;
          }
          if (!addAPIProvider(key, value)) {
            console.error(err);
          }
        } catch (e) {
          console.error(err);
        }
      }
    }
  }
}
const emptyIcon = {
  ...defaultIconProps,
  body: ""
};
const Icon = /* @__PURE__ */ defineComponent((props, { emit: emit2 }) => {
  const loader = /* @__PURE__ */ ref(null);
  function abortLoading() {
    var _a, _b;
    if (loader.value) {
      (_b = (_a = loader.value).abort) == null ? void 0 : _b.call(_a);
      loader.value = null;
    }
  }
  const rendering = /* @__PURE__ */ ref(!!props.ssr);
  const lastRenderedIconName = /* @__PURE__ */ ref("");
  const iconData = /* @__PURE__ */ shallowRef(null);
  function getIcon() {
    const icon = props.icon;
    if (typeof icon === "object" && icon !== null && typeof icon.body === "string") {
      lastRenderedIconName.value = "";
      return {
        data: icon
      };
    }
    let iconName;
    if (typeof icon !== "string" || (iconName = stringToIcon(icon, false, true)) === null) {
      return null;
    }
    let data = getIconData(iconName);
    if (!data) {
      const oldState = loader.value;
      if (!oldState || oldState.name !== icon) {
        if (data === null) {
          loader.value = {
            name: icon
          };
        } else {
          loader.value = {
            name: icon,
            abort: loadIcons([iconName], updateIconData)
          };
        }
      }
      return null;
    }
    abortLoading();
    if (lastRenderedIconName.value !== icon) {
      lastRenderedIconName.value = icon;
      nextTick(() => {
        emit2("load", icon);
      });
    }
    const customise = props.customise;
    if (customise) {
      data = Object.assign({}, data);
      const customised = customise(data.body, iconName.name, iconName.prefix, iconName.provider);
      if (typeof customised === "string") {
        data.body = customised;
      }
    }
    const classes = ["iconify"];
    if (iconName.prefix !== "") {
      classes.push("iconify--" + iconName.prefix);
    }
    if (iconName.provider !== "") {
      classes.push("iconify--" + iconName.provider);
    }
    return { data, classes };
  }
  function updateIconData() {
    var _a;
    const icon = getIcon();
    if (!icon) {
      iconData.value = null;
    } else if (icon.data !== ((_a = iconData.value) == null ? void 0 : _a.data)) {
      iconData.value = icon;
    }
  }
  if (rendering.value) {
    updateIconData();
  } else {
    onMounted(() => {
      rendering.value = true;
      updateIconData();
    });
  }
  watch(() => props.icon, updateIconData);
  onUnmounted(abortLoading);
  return () => {
    const icon = iconData.value;
    if (!icon) {
      return render(emptyIcon, props);
    }
    let newProps = props;
    if (icon.classes) {
      newProps = {
        ...props,
        class: icon.classes.join(" ")
      };
    }
    return render({
      ...defaultIconProps,
      ...icon.data
    }, newProps);
  };
}, {
  props: [
    // Icon and render mode
    "icon",
    "mode",
    "ssr",
    // Layout and style
    "width",
    "height",
    "style",
    "color",
    "inline",
    // Transformations
    "rotate",
    "hFlip",
    "horizontalFlip",
    "vFlip",
    "verticalFlip",
    "flip",
    // Misc
    "id",
    "ariaHidden",
    "customise",
    "title"
  ],
  emits: ["load"]
});
const __vite_import_meta_env__$1 = {};
let cached = null;
function readRunMode() {
  var _a;
  if (typeof import.meta !== "undefined" && __vite_import_meta_env__$1 && "web") {
    return "web";
  }
  if (typeof window !== "undefined" && ((_a = window.__APP_RUNTIME__) == null ? void 0 : _a.runMode)) {
    return window.__APP_RUNTIME__.runMode;
  }
  return "web";
}
function readNeedAuth() {
  var _a;
  if (typeof window !== "undefined" && typeof ((_a = window.__APP_RUNTIME__) == null ? void 0 : _a.needAuth) === "boolean") {
    return window.__APP_RUNTIME__.needAuth;
  }
  return true;
}
function readAppName() {
  var _a;
  if (typeof window !== "undefined" && ((_a = window.__APP_RUNTIME__) == null ? void 0 : _a.appName)) {
    return window.__APP_RUNTIME__.appName;
  }
  return "";
}
function getRuntime() {
  if (cached) return cached;
  cached = {
    runMode: readRunMode(),
    // undefined / null 都视为 true,这是"安全默认"
    needAuth: readNeedAuth(),
    appName: readAppName()
  };
  return cached;
}
let resolvedBaseURL = null;
async function detectDesktopBaseURL() {
  var _a, _b, _c, _d;
  try {
    const port = (_d = (_c = (_b = (_a = window == null ? void 0 : window.go) == null ? void 0 : _a.app) == null ? void 0 : _b.AppService) == null ? void 0 : _c.GetServerPort) == null ? void 0 : _d.call(_c);
    if (port && port > 0) {
      return `http://127.0.0.1:${port}`;
    }
  } catch (e) {
  }
  const origin = window.location.origin;
  try {
    const resp = await fetch(`${origin}/api/health`, { method: "GET" });
    if (resp.ok) {
      return origin;
    }
  } catch (e) {
  }
  return origin;
}
async function resolveBaseURL() {
  if (resolvedBaseURL !== null) return resolvedBaseURL;
  const proto = window.location.protocol;
  const runMode2 = getRuntime().runMode;
  if (proto === "http:" || proto === "https:") {
    if (runMode2 === "desktop") {
      resolvedBaseURL = await detectDesktopBaseURL();
    } else {
      resolvedBaseURL = "";
    }
  } else {
    resolvedBaseURL = "";
  }
  return resolvedBaseURL;
}
const http$1 = {
  async request(method, path, body) {
    const base = await resolveBaseURL();
    const url = `${base}${path}`;
    const opts = {
      method,
      headers: { "Content-Type": "application/json" }
    };
    if (body !== void 0) {
      opts.body = typeof body === "string" ? body : JSON.stringify(body);
    }
    const resp = await fetch(url, opts);
    const text = await resp.text();
    let data;
    try {
      data = text ? JSON.parse(text) : null;
    } catch (e) {
      data = text;
    }
    if (!resp.ok) {
      const err = new Error(`HTTP ${resp.status} ${resp.statusText}`);
      err.status = resp.status;
      err.data = data;
      throw err;
    }
    return data;
  },
  get(path) {
    return this.request("GET", path);
  },
  post(path, body) {
    return this.request("POST", path, body ?? {});
  },
  put(path, body) {
    return this.request("PUT", path, body ?? {});
  },
  delete(path) {
    return this.request("DELETE", path);
  }
};
class ApiError extends Error {
  constructor({ message, status = 0, code: code2 = null, data = null } = {}) {
    super(message || "api request failed");
    this.name = "ApiError";
    this.status = status;
    this.code = code2;
    this.data = data;
  }
}
class HttpError extends ApiError {
  constructor(opts = {}) {
    super({ ...opts, message: opts.message || `http error: ${opts.status || "network"}` });
    this.name = "HttpError";
  }
}
class BusinessError extends ApiError {
  constructor(opts = {}) {
    super({
      ...opts,
      // 业务失败时 status 通常是 200,只把 msg 当 message
      message: opts.message || opts.msg || "business error"
    });
    this.name = "BusinessError";
    this.msg = opts.msg || opts.message || "";
  }
}
class TimeoutError extends HttpError {
  constructor(opts = {}) {
    super({ ...opts, status: opts.status || 0, message: opts.message || "request timeout" });
    this.name = "TimeoutError";
  }
}
const DEFAULT_TIMEOUT_MS = 15e3;
async function readBody(resp) {
  const text = await resp.text();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch (_) {
    return text;
  }
}
async function request$1(config) {
  const {
    url,
    method = "GET",
    headers = {},
    body,
    timeout = DEFAULT_TIMEOUT_MS,
    signal
  } = config || {};
  const controller = new AbortController();
  const onExternalAbort = () => controller.abort(signal == null ? void 0 : signal.reason);
  if (signal) {
    if (signal.aborted) {
      controller.abort(signal.reason);
    } else {
      signal.addEventListener("abort", onExternalAbort, { once: true });
    }
  }
  let timer = null;
  if (timeout > 0) {
    timer = setTimeout(() => controller.abort(new DOMException("Timeout", "TimeoutError")), timeout);
  }
  try {
    const resp = await fetch(url, {
      method,
      headers,
      body,
      signal: controller.signal,
      // 不让浏览器自动 follow redirect 时丢 headers;这里显式保留 credentials 关闭(走 same-origin)
      credentials: "same-origin"
    });
    const data = await readBody(resp);
    const result = {
      ok: resp.ok,
      status: resp.status,
      statusText: resp.statusText,
      headers: resp.headers,
      data
    };
    if (!resp.ok) {
      const message = data && typeof data === "object" && (data.error || data.message || data.msg) || `HTTP ${resp.status} ${resp.statusText}`;
      throw new HttpError({ message, status: resp.status, data });
    }
    return result;
  } catch (err) {
    if (err && err.name === "AbortError" && !(signal == null ? void 0 : signal.aborted)) {
      throw new TimeoutError({ message: `request timeout after ${timeout}ms` });
    }
    if (err instanceof HttpError) throw err;
    if (err && err.name === "TimeoutError") {
      throw new TimeoutError({ message: err.message || "request timeout" });
    }
    throw new HttpError({ message: (err == null ? void 0 : err.message) || "network error", status: 0 });
  } finally {
    if (timer) clearTimeout(timer);
    if (signal) signal.removeEventListener("abort", onExternalAbort);
  }
}
function createManager() {
  const handlers = [];
  return {
    use(onFulfilled, onRejected) {
      const id = handlers.length;
      handlers.push({ onFulfilled, onRejected });
      return id;
    },
    eject(id) {
      handlers[id] = null;
    },
    /**
     * 串联运行所有 handler。
     * - 第一个 handler 接收 initial
     * - 每个 handler 返回值传给下一个
     * - 任一 handler 抛错或 reject,跳到对应 onRejected;后续 onFulfilled 跳过
     */
    async run(initial) {
      let value = initial;
      for (const h2 of handlers) {
        if (!h2) continue;
        if (!h2.onFulfilled) continue;
        try {
          value = await h2.onFulfilled(value);
        } catch (e) {
          if (h2.onRejected) {
            value = await h2.onRejected(e);
          } else {
            throw e;
          }
        }
      }
      return value;
    },
    /**
     * 错误链:从尾到头找 onRejected。
     */
    async runReject(err) {
      let value = err;
      for (let i = handlers.length - 1; i >= 0; i -= 1) {
        const h2 = handlers[i];
        if (!h2 || !h2.onRejected) continue;
        try {
          value = await h2.onRejected(value);
        } catch (e) {
          value = e;
        }
      }
      throw value;
    }
  };
}
const request = createManager();
const response = createManager();
const interceptors = {
  request,
  response
};
function installDefaultInterceptors() {
  request.use((cfg) => {
    if (!getRuntime().needAuth) return cfg;
    try {
      const token = typeof localStorage !== "undefined" && localStorage.getItem("token");
      if (token) {
        cfg.headers = { ...cfg.headers || {}, Authorization: `Bearer ${token}` };
      }
    } catch (_) {
    }
    return cfg;
  });
  response.use(
    null,
    async (err) => {
      if (!(err instanceof HttpError) || err.status !== 401) {
        throw err;
      }
      try {
        localStorage.removeItem("token");
      } catch (_) {
      }
      if (getRuntime().needAuth) {
        try {
          if (!window.__SKILL_BOX_401_REDIRECTING__) {
            window.__SKILL_BOX_401_REDIRECTING__ = true;
            window.location.href = "/login";
          }
        } catch (_) {
        }
      }
      throw err;
    }
  );
  response.use((resp) => {
    const data = resp && resp.data;
    if (data && typeof data === "object" && ("code" in data || "success" in data)) {
      const code2 = data.code !== void 0 ? data.code : data.success ? 1 : 0;
      if (code2 !== 1) {
        throw new BusinessError({
          status: resp.status,
          code: code2,
          msg: data.msg || data.message || "",
          data: data.data !== void 0 ? data.data : null
        });
      }
      return data.data !== void 0 ? data.data : data;
    }
    if (data && typeof data === "object") return data;
    return resp;
  });
}
let enabled = false;
function enableDebug() {
  enabled = true;
}
function disableDebug() {
  enabled = false;
}
function isDebug() {
  return enabled;
}
function dlog(...args) {
  if (!enabled) return;
  console.log("[req]", ...args);
}
function derr(...args) {
  if (!enabled) return;
  console.warn("[req]", ...args);
}
if (typeof window !== "undefined") {
  window.__skillBoxDebug = (on) => {
    if (on === void 0) on = !enabled;
    if (on) enableDebug();
    else disableDebug();
    return isDebug();
  };
}
function buildQuery(params) {
  if (!params || typeof params !== "object") return "";
  const usp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v === void 0 || v === null) continue;
    usp.set(k, typeof v === "object" ? JSON.stringify(v) : String(v));
  }
  const qs = usp.toString();
  return qs ? `?${qs}` : "";
}
async function doRequest(method, path, bodyOrParams, options = {}) {
  const {
    timeout,
    signal,
    headers,
    raw
    // raw=true 时不拼接 query(由调用方自己处理 path),也不走业务码剥离
  } = options;
  let realPath = path;
  let body;
  if (method === "GET" || method === "DELETE") {
    realPath = `${path}${buildQuery(bodyOrParams)}`;
  } else {
    body = bodyOrParams;
  }
  const base = await resolveBaseURL();
  const url = `${base}${realPath}`;
  let cfg = {
    url,
    method,
    headers: {
      "Content-Type": "application/json",
      ...headers || {}
    },
    body: body !== void 0 ? typeof body === "string" ? body : JSON.stringify(body) : void 0,
    timeout,
    signal,
    raw: !!raw
  };
  dlog(`→ ${method} ${realPath}`, { params: bodyOrParams, base });
  const t0 = Date.now();
  try {
    cfg = await interceptors.request.run(cfg);
    const resp = await request$1(cfg);
    const latency = Date.now() - t0;
    dlog(`<- ${method} ${realPath} ${resp.status} (${latency}ms)`, resp.data);
    if (cfg.raw) return resp;
    const data = await interceptors.response.run(resp);
    dlog(`[ok] ${method} ${realPath} resolved`, data);
    return data;
  } catch (err) {
    const latency = Date.now() - t0;
    derr(`[err] ${method} ${realPath} (${latency}ms)`, err && err.message, err);
    throw err;
  }
}
const http = {
  get(path, params, options) {
    return doRequest("GET", path, params, options);
  },
  post(path, body, options) {
    return doRequest("POST", path, body, options);
  },
  put(path, body, options) {
    return doRequest("PUT", path, body, options);
  },
  delete(path, options) {
    return doRequest("DELETE", path, void 0, options);
  },
  request(method, path, bodyOrParams, options) {
    return doRequest(method.toUpperCase(), path, bodyOrParams, options);
  }
};
installDefaultInterceptors();
function listProjects(params = {}) {
  return http.get("/api/skillbox/projects", params);
}
function createProject(payload) {
  return http.post("/api/skillbox/projects/create", payload);
}
function deleteProject(id) {
  return http.post("/api/skillbox/projects/delete", { id });
}
const _export_sfc = (sfc, props) => {
  const target = sfc.__vccOpts || sfc;
  for (const [key, val] of props) {
    target[key] = val;
  }
  return target;
};
const _hoisted_1$a = {
  key: 0,
  class: "modal-header"
};
const _hoisted_2$a = { class: "modal-title" };
const _hoisted_3$9 = ["aria-label"];
const _hoisted_4$8 = { class: "modal-body" };
const _hoisted_5$8 = {
  key: 1,
  class: "modal-footer"
};
const _sfc_main$b = {
  __name: "Modal",
  props: {
    modelValue: { type: Boolean, default: false },
    title: { type: String, default: "" },
    size: { type: String, default: "md" },
    // sm | md | lg | xl | full
    maxWidth: { type: String, default: "" },
    // 覆盖 size 的最大宽度,例如 '720px'
    closeOnMask: { type: Boolean, default: true },
    closeOnEsc: { type: Boolean, default: true },
    showClose: { type: Boolean, default: true },
    // 锁定 body 滚动(默认开)
    lockScroll: { type: Boolean, default: true }
  },
  emits: ["update:modelValue", "close", "open"],
  setup(__props, { emit: __emit }) {
    const props = __props;
    const emit2 = __emit;
    const sizeMap = {
      sm: "420px",
      md: "560px",
      lg: "760px",
      xl: "960px",
      full: "min(96vw, 1200px)"
    };
    function close() {
      emit2("update:modelValue", false);
      emit2("close");
    }
    function onMaskClick() {
      if (props.closeOnMask) close();
    }
    function onKey(e) {
      if (!props.modelValue) return;
      if (e.key === "Escape" && props.closeOnEsc) close();
    }
    let savedOverflow = "";
    function setLock(lock) {
      if (!props.lockScroll) return;
      if (typeof document === "undefined") return;
      if (lock) {
        savedOverflow = document.body.style.overflow;
        document.body.style.overflow = "hidden";
      } else {
        document.body.style.overflow = savedOverflow || "";
      }
    }
    watch(
      () => props.modelValue,
      (v) => {
        if (v) {
          setLock(true);
          if (typeof window !== "undefined") window.addEventListener("keydown", onKey);
          emit2("open");
        } else {
          setLock(false);
          if (typeof window !== "undefined") window.removeEventListener("keydown", onKey);
        }
      }
    );
    onBeforeUnmount(() => {
      setLock(false);
      if (typeof window !== "undefined") window.removeEventListener("keydown", onKey);
    });
    return (_ctx, _cache) => {
      return openBlock(), createBlock(Teleport, { to: "body" }, [
        createVNode(Transition, { name: "modal" }, {
          default: withCtx(() => [
            __props.modelValue ? (openBlock(), createElementBlock("div", {
              key: 0,
              class: "modal-mask",
              onClick: withModifiers(onMaskClick, ["self"])
            }, [
              createBaseVNode("div", {
                class: "modal-container",
                style: normalizeStyle({ maxWidth: __props.maxWidth || sizeMap[__props.size] }),
                role: "dialog",
                "aria-modal": "true"
              }, [
                __props.title || __props.showClose || _ctx.$slots.header ? (openBlock(), createElementBlock("header", _hoisted_1$a, [
                  renderSlot(_ctx.$slots, "header", {}, () => [
                    createBaseVNode("h3", _hoisted_2$a, [
                      renderSlot(_ctx.$slots, "title-icon", {}, void 0, true),
                      createTextVNode(" " + toDisplayString$1(__props.title), 1)
                    ])
                  ], true),
                  __props.showClose ? (openBlock(), createElementBlock("button", {
                    key: 0,
                    class: "modal-close",
                    type: "button",
                    "aria-label": _ctx.$t ? _ctx.$t("common.close") : "Close",
                    onClick: close
                  }, [
                    createVNode(unref(Icon), {
                      icon: "mdi:close",
                      width: "18",
                      height: "18"
                    })
                  ], 8, _hoisted_3$9)) : createCommentVNode("", true)
                ])) : createCommentVNode("", true),
                createBaseVNode("div", _hoisted_4$8, [
                  renderSlot(_ctx.$slots, "default", {}, void 0, true)
                ]),
                _ctx.$slots.footer ? (openBlock(), createElementBlock("footer", _hoisted_5$8, [
                  renderSlot(_ctx.$slots, "footer", {}, void 0, true)
                ])) : createCommentVNode("", true)
              ], 4)
            ])) : createCommentVNode("", true)
          ]),
          _: 3
        })
      ]);
    };
  }
};
const Modal = /* @__PURE__ */ _export_sfc(_sfc_main$b, [["__scopeId", "data-v-d08f0bae"]]);
const _hoisted_1$9 = { class: "projects-view" };
const _hoisted_2$9 = { class: "view-header" };
const _hoisted_3$8 = { class: "view-title" };
const _hoisted_4$7 = { class: "view-icon view-icon-purple" };
const _hoisted_5$7 = { class: "toolbar" };
const _hoisted_6$7 = { class: "search-box" };
const _hoisted_7$7 = ["placeholder"];
const _hoisted_8$7 = {
  key: 0,
  class: "error-message"
};
const _hoisted_9$7 = { class: "form-grid" };
const _hoisted_10$7 = { class: "form-field" };
const _hoisted_11$7 = ["placeholder"];
const _hoisted_12$7 = { class: "form-field" };
const _hoisted_13$7 = ["placeholder"];
const _hoisted_14$7 = { class: "form-field form-field-full" };
const _hoisted_15$7 = ["placeholder"];
const _hoisted_16$6 = { class: "form-field form-field-full" };
const _hoisted_17$6 = ["placeholder"];
const _hoisted_18$6 = { class: "card" };
const _hoisted_19$6 = { class: "card-header" };
const _hoisted_20$6 = { class: "card-sub" };
const _hoisted_21$6 = {
  key: 0,
  class: "spinner"
};
const _hoisted_22$6 = { class: "table-container" };
const _hoisted_23$6 = {
  key: 0,
  class: "grid"
};
const _hoisted_24$6 = { style: { "width": "60px" } };
const _hoisted_25$6 = { style: { "width": "100px" } };
const _hoisted_26$6 = { class: "td-id" };
const _hoisted_27$6 = { class: "project-name" };
const _hoisted_28$6 = { class: "project-alias" };
const _hoisted_29$5 = { class: "td-path" };
const _hoisted_30$5 = { class: "td-desc" };
const _hoisted_31$5 = ["onClick"];
const _hoisted_32$5 = {
  key: 1,
  class: "empty-state"
};
const _hoisted_33$5 = { class: "empty-title" };
const _hoisted_34$5 = {
  key: 0,
  class: "pager"
};
const _hoisted_35$5 = ["disabled"];
const _hoisted_36$5 = { class: "pager-info" };
const _hoisted_37$5 = ["disabled"];
const _hoisted_38$5 = { class: "confirm-message" };
const _sfc_main$a = {
  __name: "ProjectsView",
  setup(__props) {
    const { t } = useI18n();
    const items = /* @__PURE__ */ ref([]);
    const total = /* @__PURE__ */ ref(0);
    const loading = /* @__PURE__ */ ref(false);
    const error = /* @__PURE__ */ ref("");
    const showForm = /* @__PURE__ */ ref(false);
    const form = /* @__PURE__ */ reactive({ name: "", alias: "", root_path: "", description: "" });
    const filter = /* @__PURE__ */ reactive({ keyword: "", page: 1, size: 10 });
    const totalPages = computed(() => Math.max(1, Math.ceil(total.value / filter.size)));
    async function reload() {
      loading.value = true;
      error.value = "";
      try {
        const resp = await listProjects({
          page: filter.page,
          size: filter.size,
          keyword: filter.keyword || void 0
        });
        items.value = (resp == null ? void 0 : resp.items) || [];
        total.value = (resp == null ? void 0 : resp.total) || 0;
      } catch (e) {
        error.value = (e == null ? void 0 : e.message) || String(e);
      } finally {
        loading.value = false;
      }
    }
    async function submit() {
      error.value = "";
      if (!form.name.trim() || !form.alias.trim() || !form.root_path.trim()) {
        error.value = t("projects.errRequired");
        return;
      }
      try {
        await createProject({ ...form });
        showForm.value = false;
        Object.assign(form, { name: "", alias: "", root_path: "", description: "" });
        filter.page = 1;
        await reload();
      } catch (e) {
        error.value = (e == null ? void 0 : e.message) || String(e);
      }
    }
    async function remove2(id) {
      const ok = await openConfirm({
        title: t("common.delete"),
        message: t("projects.confirmDelete", { id }),
        variant: "danger",
        confirmText: t("common.delete")
      });
      if (!ok) return;
      try {
        await deleteProject(id);
        await reload();
      } catch (e) {
        error.value = (e == null ? void 0 : e.message) || String(e);
      }
    }
    const confirmOpen = /* @__PURE__ */ ref(false);
    const confirmOpts = /* @__PURE__ */ reactive({
      title: "",
      message: "",
      confirmText: "",
      cancelText: "",
      variant: "default",
      resolve: null
    });
    function openConfirm(opts) {
      confirmOpts.title = opts.title || t("common.confirm");
      confirmOpts.message = opts.message || "";
      confirmOpts.confirmText = opts.confirmText || t("common.confirm");
      confirmOpts.cancelText = opts.cancelText || t("common.cancel");
      confirmOpts.variant = opts.variant || "default";
      confirmOpen.value = true;
      return new Promise((resolve2) => {
        confirmOpts.resolve = resolve2;
      });
    }
    function resolveConfirm(ok) {
      if (confirmOpts.resolve) confirmOpts.resolve(ok);
      confirmOpen.value = false;
    }
    function gotoPage(p2) {
      if (p2 < 1 || p2 > totalPages.value) return;
      filter.page = p2;
      reload();
    }
    onMounted(reload);
    return (_ctx, _cache) => {
      return openBlock(), createElementBlock("div", _hoisted_1$9, [
        createBaseVNode("header", _hoisted_2$9, [
          createBaseVNode("div", _hoisted_3$8, [
            createBaseVNode("div", _hoisted_4$7, [
              createVNode(unref(Icon), {
                icon: "mdi:folder-multiple-outline",
                width: "24",
                height: "24"
              })
            ]),
            createBaseVNode("div", null, [
              createBaseVNode("h1", null, toDisplayString$1(unref(t)("projects.title")), 1),
              createBaseVNode("p", null, toDisplayString$1(unref(t)("projects.subtitle")), 1)
            ])
          ])
        ]),
        createBaseVNode("div", _hoisted_5$7, [
          createBaseVNode("div", _hoisted_6$7, [
            createVNode(unref(Icon), {
              icon: "mdi:magnify",
              width: "16",
              height: "16",
              class: "search-icon"
            }),
            withDirectives(createBaseVNode("input", {
              "onUpdate:modelValue": _cache[0] || (_cache[0] = ($event) => filter.keyword = $event),
              placeholder: unref(t)("projects.searchPlaceholder"),
              class: "search-input",
              onKeyup: _cache[1] || (_cache[1] = withKeys(() => {
                filter.page = 1;
                reload();
              }, ["enter"]))
            }, null, 40, _hoisted_7$7), [
              [vModelText, filter.keyword]
            ])
          ]),
          createBaseVNode("button", {
            class: "primary",
            onClick: _cache[2] || (_cache[2] = ($event) => showForm.value = true)
          }, [
            createVNode(unref(Icon), {
              icon: "mdi:plus",
              width: "16",
              height: "16"
            }),
            createTextVNode(" " + toDisplayString$1(unref(t)("projects.btnNew")), 1)
          ])
        ]),
        error.value ? (openBlock(), createElementBlock("p", _hoisted_8$7, [
          createVNode(unref(Icon), {
            icon: "mdi:alert-circle-outline",
            width: "14",
            height: "14"
          }),
          createTextVNode(" " + toDisplayString$1(error.value), 1)
        ])) : createCommentVNode("", true),
        createVNode(Modal, {
          modelValue: showForm.value,
          "onUpdate:modelValue": _cache[8] || (_cache[8] = ($event) => showForm.value = $event),
          size: "md",
          title: unref(t)("projects.formTitle")
        }, {
          "title-icon": withCtx(() => [
            createVNode(unref(Icon), {
              icon: "mdi:folder-plus",
              width: "18",
              height: "18"
            })
          ]),
          footer: withCtx(() => [
            createBaseVNode("button", {
              type: "button",
              class: "ghost",
              onClick: _cache[7] || (_cache[7] = ($event) => showForm.value = false)
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:close",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("common.cancel")), 1)
            ]),
            createBaseVNode("button", {
              type: "button",
              class: "primary",
              onClick: submit
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:check",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("common.create")), 1)
            ])
          ]),
          default: withCtx(() => [
            createBaseVNode("form", {
              class: "form",
              onSubmit: withModifiers(submit, ["prevent"])
            }, [
              createBaseVNode("div", _hoisted_9$7, [
                createBaseVNode("div", _hoisted_10$7, [
                  createBaseVNode("label", null, toDisplayString$1(unref(t)("projects.name")), 1),
                  withDirectives(createBaseVNode("input", {
                    "onUpdate:modelValue": _cache[3] || (_cache[3] = ($event) => form.name = $event),
                    placeholder: unref(t)("projects.nameHint")
                  }, null, 8, _hoisted_11$7), [
                    [vModelText, form.name]
                  ])
                ]),
                createBaseVNode("div", _hoisted_12$7, [
                  createBaseVNode("label", null, toDisplayString$1(unref(t)("projects.alias")), 1),
                  withDirectives(createBaseVNode("input", {
                    "onUpdate:modelValue": _cache[4] || (_cache[4] = ($event) => form.alias = $event),
                    placeholder: unref(t)("projects.aliasHint")
                  }, null, 8, _hoisted_13$7), [
                    [vModelText, form.alias]
                  ])
                ]),
                createBaseVNode("div", _hoisted_14$7, [
                  createBaseVNode("label", null, toDisplayString$1(unref(t)("projects.rootPath")), 1),
                  withDirectives(createBaseVNode("input", {
                    "onUpdate:modelValue": _cache[5] || (_cache[5] = ($event) => form.root_path = $event),
                    placeholder: unref(t)("projects.rootPathHint")
                  }, null, 8, _hoisted_15$7), [
                    [vModelText, form.root_path]
                  ])
                ]),
                createBaseVNode("div", _hoisted_16$6, [
                  createBaseVNode("label", null, toDisplayString$1(unref(t)("projects.description")), 1),
                  withDirectives(createBaseVNode("input", {
                    "onUpdate:modelValue": _cache[6] || (_cache[6] = ($event) => form.description = $event),
                    placeholder: unref(t)("projects.descriptionHint")
                  }, null, 8, _hoisted_17$6), [
                    [vModelText, form.description]
                  ])
                ])
              ])
            ], 32)
          ]),
          _: 1
        }, 8, ["modelValue", "title"]),
        createBaseVNode("div", _hoisted_18$6, [
          createBaseVNode("header", _hoisted_19$6, [
            createBaseVNode("h3", null, [
              createVNode(unref(Icon), {
                icon: "mdi:format-list-bulleted",
                width: "16",
                height: "16"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("projects.listTitle")) + " ", 1),
              createBaseVNode("span", _hoisted_20$6, "— " + toDisplayString$1(unref(t)("common.totalCount", { count: total.value })), 1)
            ]),
            loading.value ? (openBlock(), createElementBlock("span", _hoisted_21$6)) : createCommentVNode("", true)
          ]),
          createBaseVNode("div", _hoisted_22$6, [
            items.value.length ? (openBlock(), createElementBlock("table", _hoisted_23$6, [
              createBaseVNode("thead", null, [
                createBaseVNode("tr", null, [
                  createBaseVNode("th", _hoisted_24$6, toDisplayString$1(unref(t)("projects.colId")), 1),
                  createBaseVNode("th", null, toDisplayString$1(unref(t)("projects.colName")), 1),
                  createBaseVNode("th", null, toDisplayString$1(unref(t)("projects.colAlias")), 1),
                  createBaseVNode("th", null, toDisplayString$1(unref(t)("projects.colRootPath")), 1),
                  createBaseVNode("th", null, toDisplayString$1(unref(t)("projects.colDescription")), 1),
                  createBaseVNode("th", _hoisted_25$6, toDisplayString$1(unref(t)("projects.colActions")), 1)
                ])
              ]),
              createBaseVNode("tbody", null, [
                (openBlock(true), createElementBlock(Fragment, null, renderList(items.value, (p2) => {
                  return openBlock(), createElementBlock("tr", {
                    key: p2.ID
                  }, [
                    createBaseVNode("td", _hoisted_26$6, toDisplayString$1(p2.ID), 1),
                    createBaseVNode("td", null, [
                      createBaseVNode("strong", _hoisted_27$6, toDisplayString$1(p2.Name), 1)
                    ]),
                    createBaseVNode("td", null, [
                      createBaseVNode("code", _hoisted_28$6, toDisplayString$1(p2.Alias), 1)
                    ]),
                    createBaseVNode("td", _hoisted_29$5, toDisplayString$1(p2.RootPath), 1),
                    createBaseVNode("td", _hoisted_30$5, toDisplayString$1(p2.Description || unref(t)("common.dash")), 1),
                    createBaseVNode("td", null, [
                      createBaseVNode("button", {
                        class: "action-btn action-btn-danger",
                        onClick: ($event) => remove2(p2.ID)
                      }, [
                        createVNode(unref(Icon), {
                          icon: "mdi:delete",
                          width: "12",
                          height: "12"
                        }),
                        createTextVNode(" " + toDisplayString$1(unref(t)("common.delete")), 1)
                      ], 8, _hoisted_31$5)
                    ])
                  ]);
                }), 128))
              ])
            ])) : !loading.value ? (openBlock(), createElementBlock("div", _hoisted_32$5, [
              createVNode(unref(Icon), {
                icon: "mdi:folder-open-outline",
                width: "48",
                height: "48"
              }),
              createBaseVNode("p", _hoisted_33$5, toDisplayString$1(unref(t)("projects.empty")), 1)
            ])) : createCommentVNode("", true)
          ]),
          totalPages.value > 1 ? (openBlock(), createElementBlock("footer", _hoisted_34$5, [
            createBaseVNode("button", {
              disabled: filter.page <= 1,
              onClick: _cache[9] || (_cache[9] = ($event) => gotoPage(filter.page - 1))
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:chevron-left",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("common.prev")), 1)
            ], 8, _hoisted_35$5),
            createBaseVNode("span", _hoisted_36$5, toDisplayString$1(filter.page) + " / " + toDisplayString$1(totalPages.value) + " (" + toDisplayString$1(unref(t)("common.totalCount", { count: total.value })) + ")", 1),
            createBaseVNode("button", {
              disabled: filter.page >= totalPages.value,
              onClick: _cache[10] || (_cache[10] = ($event) => gotoPage(filter.page + 1))
            }, [
              createTextVNode(toDisplayString$1(unref(t)("common.next")) + " ", 1),
              createVNode(unref(Icon), {
                icon: "mdi:chevron-right",
                width: "14",
                height: "14"
              })
            ], 8, _hoisted_37$5)
          ])) : createCommentVNode("", true)
        ]),
        createVNode(Modal, {
          modelValue: confirmOpen.value,
          "onUpdate:modelValue": _cache[13] || (_cache[13] = ($event) => confirmOpen.value = $event),
          size: "sm",
          title: confirmOpts.title,
          "close-on-mask": false
        }, {
          footer: withCtx(() => [
            createBaseVNode("button", {
              type: "button",
              class: "ghost",
              onClick: _cache[11] || (_cache[11] = ($event) => resolveConfirm(false))
            }, toDisplayString$1(confirmOpts.cancelText), 1),
            createBaseVNode("button", {
              type: "button",
              class: normalizeClass(confirmOpts.variant === "danger" ? "danger" : "primary"),
              onClick: _cache[12] || (_cache[12] = ($event) => resolveConfirm(true))
            }, toDisplayString$1(confirmOpts.confirmText), 3)
          ]),
          default: withCtx(() => [
            createBaseVNode("p", _hoisted_38$5, toDisplayString$1(confirmOpts.message), 1)
          ]),
          _: 1
        }, 8, ["modelValue", "title"])
      ]);
    };
  }
};
const ProjectsView = /* @__PURE__ */ _export_sfc(_sfc_main$a, [["__scopeId", "data-v-9ee85546"]]);
function listSkills(params = {}) {
  return http.get("/api/skillbox/skills", params);
}
function getSkill(params) {
  return http.get("/api/skillbox/skills/get", { ...params, full: params.full ? 1 : void 0 });
}
function getSkillScopeStatus(params) {
  return http.get("/api/skillbox/skills/scope-status", params);
}
function applySkill(payload) {
  return http.post("/api/skillbox/skills/apply", payload);
}
function listApplies(params) {
  return http.get("/api/skillbox/skills/apply/list", params);
}
function undoApply(payload) {
  return http.post("/api/skillbox/skills/apply/undo", payload);
}
function forceUndoApply(payload) {
  return http.post("/api/skillbox/skills/apply/force-undo", payload);
}
function createSkill(payload) {
  return http.post("/api/skillbox/skills/create", payload);
}
function updateSkill(payload) {
  return http.post("/api/skillbox/skills/update", payload);
}
function deleteSkill(payload) {
  return http.post("/api/skillbox/skills/delete", payload);
}
function runSkillTest(payload) {
  return http.post("/api/skillbox/skills/test/run", payload);
}
function createTag(payload) {
  return http.post("/api/skillbox/skills/tags/create", payload);
}
function listTags(params = {}) {
  return http.get("/api/skillbox/skills/tags/list", params);
}
function deleteTag(payload) {
  return http.post("/api/skillbox/skills/tags/delete", payload);
}
function diffTag(params = {}) {
  return http.get("/api/skillbox/skills/tags/diff", params);
}
function rollbackTag(payload) {
  return http.post("/api/skillbox/skills/tags/rollback", payload);
}
function listPresets() {
  return http.get("/api/skillbox/ai/presets");
}
async function parseSSE(resp, { onEvent, onDone, onError }) {
  if (!resp.body || !resp.body.getReader) {
    onError && onError(new Error("stream not supported"));
    return;
  }
  const reader = resp.body.getReader();
  const decoder = new TextDecoder("utf-8");
  let buf = "";
  try {
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buf += decoder.decode(value, { stream: true });
      let idx;
      while ((idx = buf.indexOf("\n\n")) >= 0) {
        const frame = buf.slice(0, idx);
        buf = buf.slice(idx + 2);
        processFrame(frame, onEvent, onDone, onError);
        if (resp.body._done) return;
      }
    }
    if (buf.trim().length) {
      processFrame(buf, onEvent, onDone, onError);
    }
  } catch (e) {
    onError && onError(e);
  } finally {
    reader.releaseLock();
  }
}
function processFrame(frame, onEvent, onDone, onError) {
  const lines = frame.split("\n");
  const dataLines = [];
  for (const line of lines) {
    if (!line) continue;
    if (line.startsWith(":")) continue;
    if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trimStart());
    }
  }
  if (!dataLines.length) return;
  const data = dataLines.join("\n");
  if (data === "[DONE]") {
    onDone && onDone();
    return;
  }
  try {
    const obj = JSON.parse(data);
    onEvent && onEvent(obj);
  } catch (e) {
    onError && onError(new Error(`bad sse frame: ${data}`));
  }
}
async function chatStream(body, { onEvent, onDone, onError, onOpen } = {}) {
  const ctrl = new AbortController();
  let resp;
  try {
    resp = await fetch("/api/skillbox/ai/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      signal: ctrl.signal
    });
  } catch (e) {
    onError && onError(e);
    return { abort: () => ctrl.abort() };
  }
  if (!resp.ok) {
    let msg = `http ${resp.status}`;
    try {
      const j = await resp.json();
      msg = j.error || msg;
    } catch (_) {
    }
    onError && onError(new Error(msg));
    return { abort: () => ctrl.abort() };
  }
  if (onOpen) onOpen();
  await parseSSE(resp, {
    onEvent,
    onDone: () => {
      resp.body && (resp.body._done = true);
      onDone && onDone();
    },
    onError
  });
  return { abort: () => ctrl.abort() };
}
const _hoisted_1$8 = { class: "ai-panel" };
const _hoisted_2$8 = { class: "ai-header" };
const _hoisted_3$7 = ["title"];
const _hoisted_4$6 = { class: "presets" };
const _hoisted_5$6 = ["title", "onClick"];
const _hoisted_6$6 = {
  key: 0,
  class: "hint"
};
const _hoisted_7$6 = {
  key: 0,
  class: "empty"
};
const _hoisted_8$6 = { class: "meta" };
const _hoisted_9$6 = { class: "body" };
const _hoisted_10$6 = {
  key: 0,
  class: "cursor"
};
const _hoisted_11$6 = ["onClick"];
const _hoisted_12$6 = { class: "composer" };
const _hoisted_13$6 = ["placeholder", "disabled", "onKeydown"];
const _hoisted_14$6 = { class: "actions" };
const _hoisted_15$6 = ["disabled"];
const _sfc_main$9 = {
  __name: "AIPanel",
  props: {
    contextText: { type: String, default: "" },
    provider: { type: String, default: "" }
  },
  emits: ["apply", "error"],
  setup(__props, { emit: __emit }) {
    const { t } = useI18n();
    const props = __props;
    const emit2 = __emit;
    const presets = /* @__PURE__ */ ref([]);
    const activePreset = /* @__PURE__ */ ref(null);
    const messages = /* @__PURE__ */ ref([]);
    const input = /* @__PURE__ */ ref("");
    const busy = /* @__PURE__ */ ref(false);
    const historyEl = /* @__PURE__ */ ref(null);
    const abortRef = /* @__PURE__ */ ref(null);
    async function loadPresets() {
      try {
        const resp = await listPresets();
        presets.value = (resp == null ? void 0 : resp.items) || [];
      } catch (e) {
        presets.value = [];
      }
    }
    onMounted(loadPresets);
    function pushMsg(role, text, extra = {}) {
      messages.value.push({ role, text, ...extra });
      nextTick(scrollToBottom);
    }
    function scrollToBottom() {
      const el = historyEl.value;
      if (el) el.scrollTop = el.scrollHeight;
    }
    function pickPreset(p2) {
      activePreset.value = p2;
      if (p2.id === "find_duplicates") {
        pushMsg("assistant", t("skills.ai.pickedDedupe"));
      } else {
        pushMsg("assistant", t("skills.ai.pickedPreset", { title: p2.title, description: p2.description }));
      }
    }
    function buildVars() {
      var _a;
      const vars = {};
      if (props.contextText) vars.skill_md = props.contextText;
      if (((_a = activePreset.value) == null ? void 0 : _a.id) === "find_duplicates") {
        vars.skill_list = input.value || props.contextText;
      } else {
        vars.skill_md = props.contextText || input.value;
      }
      return vars;
    }
    async function send2() {
      if (busy.value) return;
      if (!activePreset.value) {
        pushMsg("assistant", t("skills.ai.pickFirst"));
        return;
      }
      const userText = activePreset.value.id === "find_duplicates" ? input.value || "" : input.value || t("skills.ai.noExtraInput");
      pushMsg("user", userText);
      input.value;
      input.value = "";
      busy.value = true;
      const placeholderIdx = messages.value.length;
      pushMsg("assistant", "", { pending: true });
      let buf = "";
      let finished = false;
      const onEvent = (ev) => {
        if (ev.kind === "chunk") {
          buf += ev.text || "";
          messages.value[placeholderIdx].text = buf;
          nextTick(scrollToBottom);
        } else if (ev.kind === "error") {
          finished = true;
          messages.value[placeholderIdx].text = buf + `

` + t("skills.ai.errorTag", { msg: ev.err || "unknown" });
          messages.value[placeholderIdx].pending = false;
          busy.value = false;
          emit2("error", ev.err);
        } else if (ev.kind === "done") ;
      };
      const onDone = () => {
        finished = true;
        messages.value[placeholderIdx].pending = false;
        busy.value = false;
        if (buf && activePreset.value.id === "optimize_frontmatter") {
          emit2("apply", buf);
        }
      };
      const onError = (err) => {
        if (finished) return;
        finished = true;
        messages.value[placeholderIdx].text = (buf || "") + `

` + t("skills.ai.errorTag", { msg: (err == null ? void 0 : err.message) || err });
        messages.value[placeholderIdx].pending = false;
        busy.value = false;
        emit2("error", (err == null ? void 0 : err.message) || err);
      };
      abortRef.value = await chatStream(
        {
          provider: props.provider,
          preset_id: activePreset.value.id,
          vars: buildVars()
        },
        { onEvent, onDone, onError }
      );
    }
    function stop() {
      var _a;
      if ((_a = abortRef.value) == null ? void 0 : _a.abort) abortRef.value.abort();
      busy.value = false;
    }
    function clear() {
      messages.value = [];
      input.value = "";
    }
    function copy(text) {
      var _a;
      (_a = navigator.clipboard) == null ? void 0 : _a.writeText(text || "");
    }
    return (_ctx, _cache) => {
      const _component_Icon = resolveComponent("Icon");
      return openBlock(), createElementBlock("aside", _hoisted_1$8, [
        createBaseVNode("header", _hoisted_2$8, [
          createBaseVNode("strong", null, [
            createVNode(_component_Icon, {
              icon: "mdi:robot",
              width: "14",
              height: "14",
              class: "ai-icon"
            }),
            createTextVNode(" " + toDisplayString$1(unref(t)("skills.ai.header")), 1)
          ]),
          createBaseVNode("button", {
            class: "link",
            onClick: clear,
            title: unref(t)("skills.ai.clear")
          }, toDisplayString$1(unref(t)("skills.ai.clear")), 9, _hoisted_3$7)
        ]),
        createBaseVNode("div", _hoisted_4$6, [
          (openBlock(true), createElementBlock(Fragment, null, renderList(presets.value, (p2) => {
            var _a;
            return openBlock(), createElementBlock("button", {
              key: p2.id,
              class: normalizeClass(["chip", { active: ((_a = activePreset.value) == null ? void 0 : _a.id) === p2.id }]),
              title: p2.description,
              onClick: ($event) => pickPreset(p2)
            }, toDisplayString$1(p2.title), 11, _hoisted_5$6);
          }), 128)),
          !presets.value.length ? (openBlock(), createElementBlock("span", _hoisted_6$6, toDisplayString$1(unref(t)("skills.ai.hintNoProvider")), 1)) : createCommentVNode("", true)
        ]),
        createBaseVNode("div", {
          class: "history",
          ref_key: "historyEl",
          ref: historyEl
        }, [
          !messages.value.length ? (openBlock(), createElementBlock("p", _hoisted_7$6, [
            createVNode(_component_Icon, {
              icon: "mdi:chat-outline",
              width: "24",
              height: "24"
            }),
            createBaseVNode("span", null, toDisplayString$1(unref(t)("skills.ai.empty")), 1)
          ])) : createCommentVNode("", true),
          (openBlock(true), createElementBlock(Fragment, null, renderList(messages.value, (m, i) => {
            return openBlock(), createElementBlock("article", {
              key: i,
              class: normalizeClass(["msg", ["role-" + m.role, { pending: m.pending }]])
            }, [
              createBaseVNode("div", _hoisted_8$6, toDisplayString$1(m.role === "user" ? unref(t)("skills.ai.roleUser") : unref(t)("skills.ai.roleAssistant")), 1),
              createBaseVNode("pre", _hoisted_9$6, [
                createTextVNode(toDisplayString$1(m.text), 1),
                m.pending ? (openBlock(), createElementBlock("span", _hoisted_10$6, "▍")) : createCommentVNode("", true)
              ]),
              !m.pending && m.text ? (openBlock(), createElementBlock("button", {
                key: 0,
                class: "link small",
                onClick: ($event) => copy(m.text)
              }, toDisplayString$1(unref(t)("skills.ai.copy")), 9, _hoisted_11$6)) : createCommentVNode("", true)
            ], 2);
          }), 128))
        ], 512),
        createBaseVNode("footer", _hoisted_12$6, [
          withDirectives(createBaseVNode("textarea", {
            "onUpdate:modelValue": _cache[0] || (_cache[0] = ($event) => input.value = $event),
            placeholder: activePreset.value ? unref(t)("skills.ai.inputPlaceholderHint") : unref(t)("skills.ai.inputPlaceholderNoPreset"),
            disabled: !activePreset.value,
            rows: "3",
            onKeydown: [
              withKeys(withModifiers(send2, ["meta", "prevent"]), ["enter"]),
              withKeys(withModifiers(send2, ["ctrl", "prevent"]), ["enter"])
            ]
          }, null, 40, _hoisted_13$6), [
            [vModelText, input.value]
          ]),
          createBaseVNode("div", _hoisted_14$6, [
            busy.value ? (openBlock(), createElementBlock("button", {
              key: 0,
              class: "danger",
              onClick: stop
            }, [
              createVNode(_component_Icon, {
                icon: "mdi:stop",
                width: "12",
                height: "12"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("skills.ai.stop")), 1)
            ])) : (openBlock(), createElementBlock("button", {
              key: 1,
              class: "primary",
              disabled: !activePreset.value,
              onClick: send2
            }, [
              createVNode(_component_Icon, {
                icon: "mdi:send",
                width: "12",
                height: "12"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("skills.ai.send")), 1)
            ], 8, _hoisted_15$6))
          ])
        ])
      ]);
    };
  }
};
const AIPanel = /* @__PURE__ */ _export_sfc(_sfc_main$9, [["__scopeId", "data-v-8e44b9e9"]]);
function escapeHtml(s) {
  return String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
}
function escapeAttr(s) {
  return escapeHtml(s);
}
function renderInline(text) {
  const codeStash = [];
  text = text.replace(/`([^`]+)`/g, (_, code2) => {
    const i = codeStash.length;
    codeStash.push(`<code>${escapeHtml(code2)}</code>`);
    return `\0CODE${i}\0`;
  });
  text = text.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (_, label, url) => {
    const safeUrl = /^(https?:|mailto:|file:|\/|#)/i.test(url) ? url : "#";
    return `<a href="${escapeAttr(safeUrl)}" target="_blank" rel="noopener noreferrer">${label}</a>`;
  });
  text = text.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
  text = text.replace(/(^|[^*])\*([^*]+)\*/g, "$1<em>$2</em>");
  text = text.replace(/\u0000CODE(\d+)\u0000/g, (_, i) => codeStash[Number(i)]);
  return text;
}
function renderTableRow(cells, isHeader) {
  const tag = isHeader ? "th" : "td";
  const html = cells.map((c) => `<${tag}>${renderInline(c.trim())}</${tag}>`).join("");
  return `<tr>${html}</tr>`;
}
function renderMarkdown(src) {
  if (!src) return "";
  const lines = String(src).replace(/\r\n?/g, "\n").split("\n");
  const out = [];
  let i = 0;
  let inList = null;
  let inCode = false;
  let codeBuf = [];
  let codeLang = "";
  function closeList() {
    if (inList) {
      out.push(`</${inList}>`);
      inList = null;
    }
  }
  while (i < lines.length) {
    const rawLine = lines[i];
    const fence = rawLine.match(/^```(\s*[\w+-]*)?\s*$/);
    if (fence) {
      if (!inCode) {
        closeList();
        inCode = true;
        codeBuf = [];
        codeLang = (fence[1] || "").trim();
      } else {
        const lang = codeLang ? ` data-lang="${escapeAttr(codeLang)}"` : "";
        out.push(`<pre${lang}><code>${escapeHtml(codeBuf.join("\n"))}</code></pre>`);
        inCode = false;
        codeBuf = [];
        codeLang = "";
      }
      i++;
      continue;
    }
    if (inCode) {
      codeBuf.push(rawLine);
      i++;
      continue;
    }
    const line = rawLine;
    const h2 = line.match(/^(#{1,6})\s+(.*)$/);
    if (h2) {
      closeList();
      const level = h2[1].length;
      out.push(`<h${level}>${renderInline(escapeHtml(h2[2].trim()))}</h${level}>`);
      i++;
      continue;
    }
    if (/^---+\s*$/.test(line) || /^\*\*\*+\s*$/.test(line)) {
      closeList();
      out.push("<hr/>");
      i++;
      continue;
    }
    if (/^>\s?/.test(line)) {
      closeList();
      const buf2 = [];
      while (i < lines.length && /^>\s?/.test(lines[i])) {
        buf2.push(lines[i].replace(/^>\s?/, ""));
        i++;
      }
      out.push(`<blockquote>${renderInline(escapeHtml(buf2.join(" ")))}</blockquote>`);
      continue;
    }
    const ul = line.match(/^[-*+]\s+(.*)$/);
    const ol = line.match(/^\d+\.\s+(.*)$/);
    if (ul || ol) {
      const kind = ul ? "ul" : "ol";
      if (inList && inList !== kind) closeList();
      if (!inList) {
        out.push(`<${kind}>`);
        inList = kind;
      }
      out.push(`<li>${renderInline(escapeHtml((ul ? ul[1] : ol[1]).trim()))}</li>`);
      i++;
      continue;
    }
    if (inList && line.trim() === "") {
      if (i + 1 < lines.length && (/^[-*+]\s+/.test(lines[i + 1]) || /^\d+\.\s+/.test(lines[i + 1]))) {
        i++;
        continue;
      }
      closeList();
      i++;
      continue;
    }
    if (/^\s*\|.*\|\s*$/.test(line)) {
      closeList();
      const headerCells = line.trim().replace(/^\||\|$/g, "").split("|");
      if (i + 1 < lines.length && /^\s*\|?[\s:|-]+\|?\s*$/.test(lines[i + 1])) {
        i += 2;
        const rows = [];
        while (i < lines.length && /^\s*\|.*\|\s*$/.test(lines[i])) {
          const cells = lines[i].trim().replace(/^\||\|$/g, "").split("|");
          rows.push(cells);
          i++;
        }
        out.push("<table>");
        out.push("<thead>");
        out.push(renderTableRow(headerCells, true));
        out.push("</thead>");
        if (rows.length) {
          out.push("<tbody>");
          for (const r of rows) out.push(renderTableRow(r, false));
          out.push("</tbody>");
        }
        out.push("</table>");
        continue;
      }
    }
    if (line.trim() === "") {
      closeList();
      i++;
      continue;
    }
    closeList();
    const buf = [line];
    i++;
    while (i < lines.length) {
      const nx = lines[i];
      if (nx.trim() === "") break;
      if (/^#{1,6}\s+/.test(nx)) break;
      if (/^[-*+]\s+/.test(nx)) break;
      if (/^\d+\.\s+/.test(nx)) break;
      if (/^```/.test(nx)) break;
      if (/^>\s?/.test(nx)) break;
      if (/^\s*\|.*\|\s*$/.test(nx)) break;
      buf.push(nx);
      i++;
    }
    out.push(`<p>${renderInline(escapeHtml(buf.join(" ")))}</p>`);
  }
  if (inCode) {
    out.push(`<pre><code>${escapeHtml(codeBuf.join("\n"))}</code></pre>`);
  }
  closeList();
  return out.join("\n");
}
const __vite_import_meta_env__ = {};
function resolveRunMode() {
  var _a;
  if (typeof import.meta !== "undefined" && __vite_import_meta_env__ && "web") {
    return "web";
  }
  if (typeof window !== "undefined" && ((_a = window.__APP_RUNTIME__) == null ? void 0 : _a.runMode)) {
    return window.__APP_RUNTIME__.runMode;
  }
  try {
    return getRuntime().runMode;
  } catch (_) {
    return "web";
  }
}
const runMode = resolveRunMode();
const isDesktop = runMode === "desktop";
function guard(name, fn) {
  return async (...args) => {
    if (!isDesktop) {
      throw new Error(`desktop capability "${name}" unavailable: not running in desktop mode`);
    }
    return fn(...args);
  };
}
function createEventSubscriber() {
  return { subscribe: () => () => {
  } };
}
const events = createEventSubscriber();
function createWebPlatform() {
  return {
    isDesktop: false,
    runMode: "web",
    app: {
      async getVersion() {
        return "web";
      },
      async getServerPort() {
        return 0;
      },
      async health() {
        return "web";
      },
      async quit() {
      }
    },
    window: {
      async toggleAlwaysOnTop() {
        return false;
      },
      async show() {
      },
      async toggleMaximise() {
      }
    },
    platform: {
      os: () => "web",
      arch: () => "web",
      async clipboardText() {
        return "";
      },
      async setClipboardText() {
        return false;
      },
      async openExternal(url) {
        window.open(url, "_blank", "noopener");
      }
    },
    fs: {
      // 读本地文件文本(Web 端走后端 HTTP,fsutil 兜底处理;失败抛错给调用方)
      async readText(path) {
        try {
          const r = await http$1.post("/api/desktop/fs/read-text", { path });
          return (r == null ? void 0 : r.content) || "";
        } catch (e) {
          throw new Error(`readText(${path}) failed: ${(e == null ? void 0 : e.message) || e}`);
        }
      },
      // reveal 在系统文件管理器显示该路径。
      // Web 端桌面 hook 不存在 → 501 带回退 URL(父目录 file://),用 openExternal 打开。
      async reveal(path) {
        var _a, _b, _c;
        try {
          await http$1.post("/api/desktop/fs/reveal", { path });
          return true;
        } catch (e) {
          const fb = ((_a = e == null ? void 0 : e.data) == null ? void 0 : _a.fallback_url) || ((_c = (_b = e == null ? void 0 : e.response) == null ? void 0 : _b.data) == null ? void 0 : _c.fallback_url);
          if (fb) {
            window.open(fb, "_blank", "noopener");
            return true;
          }
          throw new Error(`reveal(${path}) failed: ${(e == null ? void 0 : e.message) || e}`);
        }
      }
    },
    notify: {
      async hasPermission() {
        return false;
      },
      async requestPermission() {
        return false;
      },
      async show() {
        return false;
      },
      onResult() {
        return () => {
        };
      }
    },
    shortcut: {
      async register() {
        return false;
      },
      async unregister() {
        return false;
      },
      async list() {
        return [];
      }
    },
    prefs: {
      // web 端 prefs 用 /api/user/prefs 之类的业务路由(暂未实现),此处返回空
      async get() {
        return ["", false];
      },
      async set() {
        return false;
      },
      async getAll() {
        return {};
      }
    }
  };
}
function createDesktopPlatform() {
  return {
    isDesktop: true,
    runMode: "desktop",
    app: {
      // 桌面 webview 直接 load 后端 URL,不需要单独拿 port;返回 0 让 baseURL 解析走相对路径
      async getVersion() {
        try {
          return await http$1.get("/api/desktop/app/version");
        } catch (_) {
          return "desktop";
        }
      },
      async getServerPort() {
        return 0;
      },
      async health() {
        try {
          return await http$1.get("/api/desktop/app/health");
        } catch (_) {
          return "unavailable";
        }
      },
      async quit() {
        try {
          await http$1.post("/api/desktop/app/quit", {});
        } catch (_) {
        }
      }
    },
    window: {
      toggleAlwaysOnTop: guard("window.toggleAlwaysOnTop", () => http$1.post("/api/desktop/window/toggle-always-on-top", {})),
      show: guard("window.show", () => http$1.post("/api/desktop/window/show", {})),
      toggleMaximise: guard("window.toggleMaximise", () => http$1.post("/api/desktop/window/toggle-maximise", {}))
    },
    platform: {
      os: () => {
        var _a;
        try {
          return ((_a = window.__APP_RUNTIME__) == null ? void 0 : _a.os) || "desktop";
        } catch (_) {
          return "desktop";
        }
      },
      arch: () => {
        var _a;
        try {
          return ((_a = window.__APP_RUNTIME__) == null ? void 0 : _a.arch) || "desktop";
        } catch (_) {
          return "desktop";
        }
      },
      clipboardText: guard("platform.clipboardText", () => http$1.get("/api/desktop/clipboard/text")),
      setClipboardText: guard("platform.setClipboardText", (text) => http$1.put("/api/desktop/clipboard/text", { text })),
      openExternal: guard("platform.openExternal", (url) => http$1.post("/api/desktop/open-external", { url }))
    },
    fs: {
      async readText(path) {
        const r = await http$1.post("/api/desktop/fs/read-text", { path });
        return (r == null ? void 0 : r.content) || "";
      },
      async reveal(path) {
        var _a, _b, _c;
        try {
          await http$1.post("/api/desktop/fs/reveal", { path });
          return true;
        } catch (e) {
          const fb = ((_a = e == null ? void 0 : e.data) == null ? void 0 : _a.fallback_url) || ((_c = (_b = e == null ? void 0 : e.response) == null ? void 0 : _b.data) == null ? void 0 : _c.fallback_url);
          if (fb) return { ok: false, fallbackUrl: fb };
          throw e;
        }
      }
    },
    notify: {
      hasPermission: guard("notify.hasPermission", () => http$1.get("/api/desktop/notify/permission")),
      requestPermission: guard("notify.requestPermission", () => http$1.post("/api/desktop/notify/permission/request", {})),
      show: guard("notify.show", (id, title, body) => http$1.post("/api/desktop/notify/show", { id: id || "", title: title || "", body: body || "" })),
      onResult(cb) {
        return events.subscribe("notify:clicked", (actionID, notifID) => {
          try {
            cb(actionID, notifID);
          } catch (e) {
            console.error("[notify:clicked]", e);
          }
        });
      }
    },
    shortcut: {
      register: guard("shortcut.register", (combo) => http$1.post("/api/desktop/shortcut/register", { combo })),
      unregister: guard("shortcut.unregister", (combo) => http$1.post("/api/desktop/shortcut/unregister", { combo })),
      list: guard("shortcut.list", () => http$1.get("/api/desktop/shortcut/list"))
    },
    prefs: {
      // desktop 模式 prefs 走 HTTP,与业务 API 一致;
      // settings KV 由后端 bootstrap.Backend.NewSettings() 工厂方法构造,数据落 entity.Setting 表。
      // 返回值约定(对齐 wails v3 自动生成签名):
      //   get(key)      → [value: string, exists: boolean]
      //   getAll()      → { [key]: value }
      get: guard("prefs.get", async (key) => {
        const r = await http$1.get(`/api/desktop/prefs?key=${encodeURIComponent(key)}`);
        return [(r == null ? void 0 : r.value) ?? "", !!(r == null ? void 0 : r.exists)];
      }),
      set: guard("prefs.set", (key, value) => http$1.put("/api/desktop/prefs", { key, value: String(value) })),
      getAll: guard("prefs.getAll", async () => {
        const r = await http$1.get("/api/desktop/prefs");
        return (r == null ? void 0 : r.items) || {};
      })
    }
  };
}
const platform = isDesktop ? createDesktopPlatform() : createWebPlatform();
function getOnboardingStatus() {
  return http.get("/api/skillbox/onboarding/status");
}
function runOnboardingScan() {
  return http.post("/api/skillbox/onboarding/scan", {});
}
function runOnboardingImport(items = []) {
  return http.post("/api/skillbox/onboarding/import", { items });
}
const _hoisted_1$7 = {
  key: 0,
  class: "skill-title"
};
const _hoisted_2$7 = ["title"];
const _sfc_main$8 = {
  __name: "SkillTitle",
  props: {
    sourcePath: { type: String, required: true },
    fetcher: { type: Function, required: true }
  },
  setup(__props) {
    const props = __props;
    const text = /* @__PURE__ */ ref("");
    let reqId = 0;
    async function load() {
      const myId = ++reqId;
      try {
        const meta = await props.fetcher(props.sourcePath);
        if (myId !== reqId) return;
        text.value = ((meta == null ? void 0 : meta.description) || (meta == null ? void 0 : meta.title) || "").trim();
      } catch (_) {
        if (myId === reqId) text.value = "";
      }
    }
    onMounted(load);
    watch(() => props.sourcePath, load);
    return (_ctx, _cache) => {
      return text.value ? (openBlock(), createElementBlock("div", _hoisted_1$7, [
        createBaseVNode("span", {
          class: "skill-title-label",
          title: text.value
        }, toDisplayString$1(text.value), 9, _hoisted_2$7)
      ])) : createCommentVNode("", true);
    };
  }
};
const SkillTitle = /* @__PURE__ */ _export_sfc(_sfc_main$8, [["__scopeId", "data-v-08163aa6"]]);
const _hoisted_1$6 = { class: "onb" };
const _hoisted_2$6 = {
  key: 0,
  class: "message message-error"
};
const _hoisted_3$6 = {
  key: 1,
  class: "message message-success"
};
const _hoisted_4$5 = {
  key: 2,
  class: "card"
};
const _hoisted_5$5 = { class: "card-header card-header-row" };
const _hoisted_6$5 = { class: "card-sub" };
const _hoisted_7$5 = {
  key: 0,
  class: "header-actions"
};
const _hoisted_8$5 = ["disabled", "title"];
const _hoisted_9$5 = {
  key: 0,
  class: "spinner"
};
const _hoisted_10$5 = ["disabled"];
const _hoisted_11$5 = {
  key: 0,
  class: "spinner"
};
const _hoisted_12$5 = {
  key: 0,
  class: "empty-state"
};
const _hoisted_13$5 = { class: "empty-title" };
const _hoisted_14$5 = { class: "empty-hint" };
const _hoisted_15$5 = { key: 1 };
const _hoisted_16$5 = {
  class: "tool-tabs",
  role: "tablist"
};
const _hoisted_17$5 = ["aria-selected", "onClick"];
const _hoisted_18$5 = { class: "tab-name" };
const _hoisted_19$5 = { class: "tab-count" };
const _hoisted_20$5 = {
  key: 0,
  class: "tab-count-sys"
};
const _hoisted_21$5 = {
  key: 0,
  class: "tool-panel"
};
const _hoisted_22$5 = { class: "bulk-actions" };
const _hoisted_23$5 = { class: "selection-info" };
const _hoisted_24$5 = {
  key: 0,
  class: "cat-label cat-user"
};
const _hoisted_25$5 = {
  key: 1,
  class: "found-list"
};
const _hoisted_26$5 = ["checked", "disabled", "title", "onChange"];
const _hoisted_27$5 = { class: "f-main" };
const _hoisted_28$5 = { class: "f-line-1" };
const _hoisted_29$4 = { class: "f-name" };
const _hoisted_30$4 = { class: "f-ver" };
const _hoisted_31$4 = ["title"];
const _hoisted_32$4 = {
  key: 1,
  class: "f-tag f-tag-exists"
};
const _hoisted_33$4 = { class: "f-line-2" };
const _hoisted_34$4 = ["title", "onClick"];
const _hoisted_35$4 = { class: "f-path-text" };
const _hoisted_36$4 = {
  key: 2,
  class: "cat-divider"
};
const _hoisted_37$4 = { class: "cat-divider-text" };
const _hoisted_38$4 = {
  key: 3,
  class: "cat-label cat-system"
};
const _hoisted_39$4 = { class: "cat-hint" };
const _hoisted_40$4 = {
  key: 4,
  class: "found-list found-list-system"
};
const _hoisted_41$3 = { class: "found-item found-item-system" };
const _hoisted_42$3 = ["title"];
const _hoisted_43$3 = { class: "f-main" };
const _hoisted_44$3 = { class: "f-line-1" };
const _hoisted_45$3 = { class: "f-name" };
const _hoisted_46$3 = { class: "f-ver" };
const _hoisted_47$3 = ["title"];
const _hoisted_48$3 = { class: "f-line-2" };
const _hoisted_49$3 = ["title", "onClick"];
const _hoisted_50$3 = { class: "f-path-text" };
const _hoisted_51$3 = {
  key: 3,
  class: "card"
};
const _hoisted_52$3 = { class: "card-header" };
const _hoisted_53$3 = {
  key: 0,
  class: "result-stats"
};
const _hoisted_54$2 = { class: "stat-card stat-success" };
const _hoisted_55$2 = { class: "stat-number" };
const _hoisted_56$1 = { class: "stat-label" };
const _hoisted_57$1 = { class: "stat-card stat-error" };
const _hoisted_58$1 = { class: "stat-number" };
const _hoisted_59$1 = { class: "stat-label" };
const _hoisted_60$1 = { class: "stat-card" };
const _hoisted_61$1 = { class: "stat-number" };
const _hoisted_62$1 = { class: "stat-label" };
const _hoisted_63$1 = {
  key: 1,
  class: "result-list"
};
const _hoisted_64$1 = { class: "r-tool" };
const _hoisted_65$1 = { class: "r-name" };
const _hoisted_66$1 = { class: "r-msg" };
const _hoisted_67$1 = { class: "card-footer" };
const _sfc_main$7 = {
  __name: "OnboardingView",
  emits: ["done"],
  setup(__props, { emit: __emit }) {
    const { t } = useI18n();
    const emit2 = __emit;
    const phase = /* @__PURE__ */ ref("scan");
    inject("appBus", null);
    const loading = /* @__PURE__ */ ref(false);
    const error = /* @__PURE__ */ ref("");
    const success = /* @__PURE__ */ ref("");
    const adapters = /* @__PURE__ */ ref([]);
    const lastScan = /* @__PURE__ */ ref(null);
    const totalFound = /* @__PURE__ */ ref(0);
    const hasReport = /* @__PURE__ */ ref(false);
    const scanReport = /* @__PURE__ */ ref(null);
    const selected = /* @__PURE__ */ ref(/* @__PURE__ */ new Set());
    const activeToolId = /* @__PURE__ */ ref("");
    const importResult = /* @__PURE__ */ ref(null);
    const existingNames = /* @__PURE__ */ ref(/* @__PURE__ */ new Set());
    const skillTitles = /* @__PURE__ */ ref({});
    const toolIconMap = {
      claude: "mdi:robot-outline",
      codex: "mdi:cube-outline",
      cursor: "mdi:cursor-default-click-outline",
      opencode: "mdi:code-braces",
      trae: "mdi:shield-outline"
    };
    function iconOf(toolId) {
      return toolIconMap[toolId] || "mdi:puzzle-outline";
    }
    async function loadStatus() {
      loading.value = true;
      error.value = "";
      try {
        const res = await getOnboardingStatus();
        adapters.value = (res == null ? void 0 : res.adapters) || [];
        lastScan.value = (res == null ? void 0 : res.last_scan) || null;
        totalFound.value = (res == null ? void 0 : res.total_found) || 0;
        hasReport.value = !!(res == null ? void 0 : res.has_report);
      } catch (e) {
        error.value = (e == null ? void 0 : e.message) || String(e);
      } finally {
        loading.value = false;
      }
    }
    async function loadExistingNames() {
      try {
        const res = await listSkills({ page: 1, size: 1e3 });
        const set = /* @__PURE__ */ new Set();
        for (const it of (res == null ? void 0 : res.items) || []) {
          if (it == null ? void 0 : it.name) set.add(String(it.name).toLowerCase());
        }
        existingNames.value = set;
      } catch (_) {
        existingNames.value = /* @__PURE__ */ new Set();
      }
    }
    function keyOf(found) {
      return `${found.tool_id}::${found.name}@${found.version}`;
    }
    async function doScan() {
      loading.value = true;
      error.value = "";
      success.value = "";
      try {
        const [res] = await Promise.all([runOnboardingScan(), loadExistingNames()]);
        scanReport.value = res;
        selected.value = /* @__PURE__ */ new Set();
        const firstTid = (res.tools || []).find(
          (tid) => (res.found || []).some(
            (f) => f.tool_id === tid && f.category !== "system"
          )
        ) || (res.tools || [])[0];
        activeToolId.value = firstTid || "";
      } catch (e) {
        error.value = t("onboarding.errScan", { msg: (e == null ? void 0 : e.message) || e });
      } finally {
        loading.value = false;
      }
    }
    function toggleSelect(found) {
      if (isDisabled(found)) return;
      const k = keyOf(found);
      let s = new Set(selected.value);
      if (s.has(k)) {
        s.delete(k);
      } else {
        s = selectExclusiveByName(s, found);
        s.add(k);
      }
      selected.value = s;
    }
    function selectExclusiveByName(s, found) {
      const next = new Set(s);
      const nameKey = String(found.name).toLowerCase();
      for (const k of Array.from(next)) {
        const [tid, nameVer] = k.split("::");
        const [n, v] = nameVer.split("@");
        if (String(n).toLowerCase() === nameKey && v === found.version && tid !== found.tool_id) {
          next.delete(k);
        }
      }
      return next;
    }
    function isDisabled(found) {
      if (found.category === "system") return true;
      if (existingNames.value.has(String(found.name).toLowerCase())) return true;
      const wantName = String(found.name).toLowerCase();
      const wantVer = found.version;
      for (const sel of selected.value) {
        const [tid, nameVer] = sel.split("::");
        const [n, v] = nameVer.split("@");
        if (String(n).toLowerCase() === wantName && v === wantVer && tid !== found.tool_id) {
          return true;
        }
      }
      return false;
    }
    function disabledReason(found) {
      if (found.category === "system") {
        return t("onboarding.phase2.disabledSystem");
      }
      if (existingNames.value.has(String(found.name).toLowerCase())) {
        return t("onboarding.phase2.disabledExists");
      }
      return t("onboarding.phase2.disabledExclusive");
    }
    const foundByTool = computed(() => {
      var _a, _b;
      const groups = {};
      for (const tid of ((_a = scanReport.value) == null ? void 0 : _a.tools) || []) {
        groups[tid] = { name: "", items: [] };
      }
      for (const f of ((_b = scanReport.value) == null ? void 0 : _b.found) || []) {
        if (!groups[f.tool_id]) {
          groups[f.tool_id] = { name: f.tool_name || f.tool_id, items: [] };
        }
        if (!groups[f.tool_id].name) groups[f.tool_id].name = f.tool_name || f.tool_id;
        groups[f.tool_id].items.push(f);
      }
      for (const tid of Object.keys(groups)) {
        groups[tid].items.sort((a, b) => {
          const ax = a.category === "system" ? 1 : 0;
          const bx = b.category === "system" ? 1 : 0;
          if (ax !== bx) return ax - bx;
          return a.name.localeCompare(b.name);
        });
      }
      return groups;
    });
    const toolTabs = computed(
      () => Object.entries(foundByTool.value).map(([tid, g]) => ({
        toolId: tid,
        // 兜底:极端情况下 name 仍为空,用 toolId 顶上
        name: g.name || tid,
        count: g.items.filter((f) => f.category !== "system").length,
        totalCount: g.items.length,
        icon: iconOf(tid)
      }))
    );
    function selectAllInTool(tid) {
      var _a;
      let s = new Set(selected.value);
      for (const f of ((_a = foundByTool.value[tid]) == null ? void 0 : _a.items) || []) {
        if (isDisabled(f)) continue;
        s = selectExclusiveByName(s, f);
        s.add(keyOf(f));
      }
      selected.value = s;
    }
    function selectNoneInTool(tid) {
      var _a;
      const s = new Set(selected.value);
      for (const f of ((_a = foundByTool.value[tid]) == null ? void 0 : _a.items) || []) {
        if (f.category === "system") continue;
        s.delete(keyOf(f));
      }
      selected.value = s;
    }
    function selectedInTool(tid) {
      var _a;
      let n = 0;
      for (const f of ((_a = foundByTool.value[tid]) == null ? void 0 : _a.items) || []) {
        if (f.category === "system") continue;
        if (selected.value.has(keyOf(f))) n++;
      }
      return n;
    }
    function selectableInTool(tid) {
      var _a;
      let n = 0;
      for (const f of ((_a = foundByTool.value[tid]) == null ? void 0 : _a.items) || []) {
        if (!isDisabled(f)) n++;
      }
      return n;
    }
    async function fetchTitle(sourcePath) {
      if (!sourcePath) return { title: "", description: "" };
      if (sourcePath in skillTitles.value) {
        return skillTitles.value[sourcePath];
      }
      try {
        const mdPath = sourcePath.replace(/\/+$/, "") + "/SKILL.md";
        const r = await platform.fs.readText(mdPath).catch(() => null);
        const meta = r ? parseSkillMeta(r) : { title: "", description: "" };
        skillTitles.value = { ...skillTitles.value, [sourcePath]: meta };
        return meta;
      } catch (_) {
        skillTitles.value = { ...skillTitles.value, [sourcePath]: { title: "", description: "" } };
        return { title: "", description: "" };
      }
    }
    function parseSkillMeta(md) {
      const out = { title: "", description: "" };
      if (!md) return out;
      let body = md;
      if (body.startsWith("---")) {
        const end = body.indexOf("\n---", 3);
        if (end > 0) {
          const fm = body.slice(3, end);
          for (const line of fm.split("\n")) {
            const m = line.match(/^\s*description\s*:\s*(.+?)\s*$/);
            if (m) {
              let v = m[1];
              if (v.startsWith('"') && v.endsWith('"') || v.startsWith("'") && v.endsWith("'")) {
                v = v.slice(1, -1);
              }
              out.description = v;
              break;
            }
          }
          body = body.slice(end + 4);
        }
      }
      for (const line of body.split("\n")) {
        const t2 = line.trim();
        if (t2.startsWith("# ")) {
          out.title = t2.slice(2).trim();
          break;
        }
      }
      return out;
    }
    async function revealInFileManager(sourcePath) {
      if (!sourcePath) return;
      try {
        await platform.fs.reveal(sourcePath);
      } catch (_) {
        try {
          await platform.platform.openExternal("file://" + sourcePath);
        } catch (__) {
        }
      }
    }
    async function doImport() {
      loading.value = true;
      error.value = "";
      success.value = "";
      try {
        const items = Array.from(selected.value).map((k) => {
          const [tool_id, nameVer] = k.split("::");
          const [name, version2] = nameVer.split("@");
          return { tool_id, name, version: version2 };
        });
        const res = await runOnboardingImport(items);
        importResult.value = res;
        phase.value = "import";
        success.value = t("onboarding.okImport", { ok: res.ok, failed: res.failed });
        await loadStatus();
      } catch (e) {
        error.value = t("onboarding.errImport", { msg: (e == null ? void 0 : e.message) || e });
      } finally {
        loading.value = false;
      }
    }
    function reset() {
      phase.value = "scan";
      scanReport.value = null;
      importResult.value = null;
      selected.value = /* @__PURE__ */ new Set();
      activeToolId.value = "";
    }
    function goSkills() {
      emit2("done", importResult.value);
    }
    onMounted(async () => {
      await loadStatus();
      await doScan();
    });
    return (_ctx, _cache) => {
      var _a, _b, _c, _d, _e, _f, _g, _h;
      return openBlock(), createElementBlock("div", _hoisted_1$6, [
        error.value ? (openBlock(), createElementBlock("p", _hoisted_2$6, [
          createVNode(unref(Icon), {
            icon: "mdi:alert-circle-outline",
            width: "14",
            height: "14"
          }),
          createTextVNode(" " + toDisplayString$1(error.value), 1)
        ])) : createCommentVNode("", true),
        success.value ? (openBlock(), createElementBlock("p", _hoisted_3$6, [
          createVNode(unref(Icon), {
            icon: "mdi:check-circle-outline",
            width: "14",
            height: "14"
          }),
          createTextVNode(" " + toDisplayString$1(success.value), 1)
        ])) : phase.value === "scan" ? (openBlock(), createElementBlock("section", _hoisted_4$5, [
          createBaseVNode("header", _hoisted_5$5, [
            createBaseVNode("h3", null, [
              createVNode(unref(Icon), {
                icon: "mdi:folder-search",
                width: "16",
                height: "16"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("onboarding.phase2.title")) + " ", 1),
              createBaseVNode("span", _hoisted_6$5, "— " + toDisplayString$1(unref(t)("onboarding.phase2.foundSuffix", { n: ((_b = (_a = scanReport.value) == null ? void 0 : _a.found) == null ? void 0 : _b.length) || 0 })), 1)
            ]),
            ((_d = (_c = scanReport.value) == null ? void 0 : _c.found) == null ? void 0 : _d.length) ? (openBlock(), createElementBlock("div", _hoisted_7$5, [
              createBaseVNode("button", {
                class: "ghost sm",
                disabled: loading.value,
                title: unref(t)("onboarding.btnRescanTitle"),
                onClick: doScan
              }, [
                loading.value ? (openBlock(), createElementBlock("span", _hoisted_9$5)) : (openBlock(), createBlock(unref(Icon), {
                  key: 1,
                  icon: "mdi:refresh",
                  width: "14",
                  height: "14"
                })),
                createTextVNode(" " + toDisplayString$1(loading.value ? unref(t)("onboarding.btnRescanning") : unref(t)("onboarding.btnRescan")), 1)
              ], 8, _hoisted_8$5),
              createBaseVNode("button", {
                class: "primary sm",
                disabled: loading.value || selected.value.size === 0,
                onClick: doImport
              }, [
                loading.value ? (openBlock(), createElementBlock("span", _hoisted_11$5)) : (openBlock(), createBlock(unref(Icon), {
                  key: 1,
                  icon: "mdi:download",
                  width: "14",
                  height: "14"
                })),
                createTextVNode(" " + toDisplayString$1(loading.value ? unref(t)("onboarding.phase2.importing") : unref(t)("onboarding.phase2.btnImport", { n: selected.value.size })), 1)
              ], 8, _hoisted_10$5)
            ])) : createCommentVNode("", true)
          ]),
          !((_f = (_e = scanReport.value) == null ? void 0 : _e.found) == null ? void 0 : _f.length) ? (openBlock(), createElementBlock("div", _hoisted_12$5, [
            createVNode(unref(Icon), {
              icon: "mdi:magnify",
              width: "48",
              height: "48"
            }),
            createBaseVNode("p", _hoisted_13$5, toDisplayString$1(unref(t)("onboarding.phase2.empty")), 1),
            createBaseVNode("p", _hoisted_14$5, toDisplayString$1(unref(t)("onboarding.phase2.emptyHint")), 1)
          ])) : (openBlock(), createElementBlock("div", _hoisted_15$5, [
            createBaseVNode("div", _hoisted_16$5, [
              (openBlock(true), createElementBlock(Fragment, null, renderList(toolTabs.value, (tab) => {
                return openBlock(), createElementBlock("button", {
                  key: tab.toolId,
                  role: "tab",
                  "aria-selected": activeToolId.value === tab.toolId,
                  class: normalizeClass(["tool-tab", { active: activeToolId.value === tab.toolId }]),
                  onClick: ($event) => activeToolId.value = tab.toolId
                }, [
                  createVNode(unref(Icon), {
                    icon: tab.icon,
                    width: "16",
                    height: "16",
                    class: "tab-icon"
                  }, null, 8, ["icon"]),
                  createBaseVNode("span", _hoisted_18$5, toDisplayString$1(tab.name), 1),
                  createBaseVNode("span", _hoisted_19$5, [
                    createTextVNode(toDisplayString$1(tab.count), 1),
                    tab.totalCount > tab.count ? (openBlock(), createElementBlock("span", _hoisted_20$5, "+" + toDisplayString$1(tab.totalCount - tab.count), 1)) : createCommentVNode("", true)
                  ])
                ], 10, _hoisted_17$5);
              }), 128))
            ]),
            activeToolId.value && foundByTool.value[activeToolId.value] ? (openBlock(), createElementBlock("div", _hoisted_21$5, [
              createBaseVNode("div", _hoisted_22$5, [
                createBaseVNode("button", {
                  class: "sm",
                  onClick: _cache[0] || (_cache[0] = ($event) => selectAllInTool(activeToolId.value))
                }, toDisplayString$1(unref(t)("onboarding.phase2.selectAll")), 1),
                createBaseVNode("button", {
                  class: "sm ghost",
                  onClick: _cache[1] || (_cache[1] = ($event) => selectNoneInTool(activeToolId.value))
                }, toDisplayString$1(unref(t)("onboarding.phase2.selectNone")), 1),
                createBaseVNode("span", _hoisted_23$5, toDisplayString$1(unref(t)("onboarding.phase2.selected", {
                  sel: selectedInTool(activeToolId.value),
                  total: selectableInTool(activeToolId.value)
                })), 1)
              ]),
              foundByTool.value[activeToolId.value].items.some((f) => f.category === "user" || !f.category) ? (openBlock(), createElementBlock("div", _hoisted_24$5, [
                createVNode(unref(Icon), {
                  icon: "mdi:account-circle-outline",
                  width: "14",
                  height: "14"
                }),
                createTextVNode(" " + toDisplayString$1(unref(t)("onboarding.phase2.catUser")), 1)
              ])) : createCommentVNode("", true),
              foundByTool.value[activeToolId.value].items.some((f) => f.category !== "system") ? (openBlock(), createElementBlock("ul", _hoisted_25$5, [
                (openBlock(true), createElementBlock(Fragment, null, renderList(foundByTool.value[activeToolId.value].items.filter((x) => x.category !== "system"), (f) => {
                  return openBlock(), createElementBlock("li", {
                    key: keyOf(f),
                    class: normalizeClass({ selected: selected.value.has(keyOf(f)), disabled: isDisabled(f) })
                  }, [
                    createBaseVNode("label", {
                      class: normalizeClass(["found-item", { "item-disabled": isDisabled(f) }])
                    }, [
                      createBaseVNode("input", {
                        type: "checkbox",
                        checked: selected.value.has(keyOf(f)),
                        disabled: isDisabled(f),
                        title: isDisabled(f) ? disabledReason(f) : "",
                        onChange: ($event) => toggleSelect(f)
                      }, null, 40, _hoisted_26$5),
                      createBaseVNode("div", _hoisted_27$5, [
                        createBaseVNode("div", _hoisted_28$5, [
                          createBaseVNode("span", _hoisted_29$4, [
                            createBaseVNode("code", null, toDisplayString$1(f.name), 1)
                          ]),
                          createBaseVNode("span", _hoisted_30$4, "v" + toDisplayString$1(f.version), 1),
                          isDisabled(f) && f.category !== "system" ? (openBlock(), createElementBlock("span", {
                            key: 0,
                            class: "f-disabled-reason",
                            title: disabledReason(f)
                          }, [
                            createVNode(unref(Icon), {
                              icon: "mdi:block-helper",
                              width: "11",
                              height: "11"
                            }),
                            createTextVNode(" " + toDisplayString$1(disabledReason(f)), 1)
                          ], 8, _hoisted_31$4)) : createCommentVNode("", true),
                          existingNames.value.has(String(f.name).toLowerCase()) ? (openBlock(), createElementBlock("span", _hoisted_32$4, [
                            createVNode(unref(Icon), {
                              icon: "mdi:package-variant",
                              width: "11",
                              height: "11"
                            }),
                            createTextVNode(" " + toDisplayString$1(unref(t)("onboarding.phase2.tagExists")), 1)
                          ])) : createCommentVNode("", true)
                        ]),
                        createVNode(SkillTitle, {
                          "source-path": f.source_path,
                          fetcher: fetchTitle
                        }, null, 8, ["source-path"]),
                        createBaseVNode("div", _hoisted_33$4, [
                          createBaseVNode("button", {
                            type: "button",
                            class: "f-path-btn",
                            title: f.source_path,
                            onClick: withModifiers(($event) => revealInFileManager(f.source_path), ["stop"])
                          }, [
                            createVNode(unref(Icon), {
                              icon: "mdi:folder-outline",
                              width: "14",
                              height: "14"
                            })
                          ], 8, _hoisted_34$4),
                          createBaseVNode("span", _hoisted_35$4, toDisplayString$1(f.source_path), 1)
                        ])
                      ])
                    ], 2)
                  ], 2);
                }), 128))
              ])) : createCommentVNode("", true),
              foundByTool.value[activeToolId.value].items.some((f) => f.category === "system") ? (openBlock(), createElementBlock("div", _hoisted_36$4, [
                createBaseVNode("span", _hoisted_37$4, toDisplayString$1(unref(t)("onboarding.phase2.catSectionDivider")), 1)
              ])) : createCommentVNode("", true),
              foundByTool.value[activeToolId.value].items.some((f) => f.category === "system") ? (openBlock(), createElementBlock("div", _hoisted_38$4, [
                createVNode(unref(Icon), {
                  icon: "mdi:lock-outline",
                  width: "14",
                  height: "14"
                }),
                createTextVNode(" " + toDisplayString$1(unref(t)("onboarding.phase2.catSystem")) + " ", 1),
                createBaseVNode("span", _hoisted_39$4, "— " + toDisplayString$1(unref(t)("onboarding.phase2.catSystemHint")), 1)
              ])) : createCommentVNode("", true),
              foundByTool.value[activeToolId.value].items.some((f) => f.category === "system") ? (openBlock(), createElementBlock("ul", _hoisted_40$4, [
                (openBlock(true), createElementBlock(Fragment, null, renderList(foundByTool.value[activeToolId.value].items.filter((x) => x.category === "system"), (f) => {
                  return openBlock(), createElementBlock("li", {
                    key: keyOf(f),
                    class: "system-item"
                  }, [
                    createBaseVNode("span", _hoisted_41$3, [
                      createBaseVNode("input", {
                        type: "checkbox",
                        disabled: "",
                        "aria-disabled": "true",
                        title: disabledReason(f)
                      }, null, 8, _hoisted_42$3),
                      createBaseVNode("div", _hoisted_43$3, [
                        createBaseVNode("div", _hoisted_44$3, [
                          createBaseVNode("span", _hoisted_45$3, [
                            createBaseVNode("code", null, toDisplayString$1(f.name), 1)
                          ]),
                          createBaseVNode("span", _hoisted_46$3, "v" + toDisplayString$1(f.version), 1),
                          createBaseVNode("span", {
                            class: "f-disabled-reason",
                            title: disabledReason(f)
                          }, [
                            createVNode(unref(Icon), {
                              icon: "mdi:block-helper",
                              width: "11",
                              height: "11"
                            }),
                            createTextVNode(" " + toDisplayString$1(disabledReason(f)), 1)
                          ], 8, _hoisted_47$3)
                        ]),
                        createVNode(SkillTitle, {
                          "source-path": f.source_path,
                          fetcher: fetchTitle
                        }, null, 8, ["source-path"]),
                        createBaseVNode("div", _hoisted_48$3, [
                          createBaseVNode("button", {
                            type: "button",
                            class: "f-path-btn",
                            title: f.source_path,
                            onClick: withModifiers(($event) => revealInFileManager(f.source_path), ["stop"])
                          }, [
                            createVNode(unref(Icon), {
                              icon: "mdi:folder-outline",
                              width: "14",
                              height: "14"
                            })
                          ], 8, _hoisted_49$3),
                          createBaseVNode("span", _hoisted_50$3, toDisplayString$1(f.source_path), 1)
                        ])
                      ]),
                      createVNode(unref(Icon), {
                        icon: "mdi:lock-outline",
                        width: "12",
                        height: "12",
                        class: "lock-icon"
                      })
                    ])
                  ]);
                }), 128))
              ])) : createCommentVNode("", true)
            ])) : createCommentVNode("", true)
          ]))
        ])) : phase.value === "import" ? (openBlock(), createElementBlock("section", _hoisted_51$3, [
          createBaseVNode("header", _hoisted_52$3, [
            createBaseVNode("h3", null, [
              createVNode(unref(Icon), {
                icon: "mdi:check-circle",
                width: "16",
                height: "16"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("onboarding.phase3.title")), 1)
            ])
          ]),
          importResult.value ? (openBlock(), createElementBlock("div", _hoisted_53$3, [
            createBaseVNode("div", _hoisted_54$2, [
              createBaseVNode("span", _hoisted_55$2, toDisplayString$1(importResult.value.ok), 1),
              createBaseVNode("span", _hoisted_56$1, toDisplayString$1(unref(t)("onboarding.phase3.statOk")), 1)
            ]),
            createBaseVNode("div", _hoisted_57$1, [
              createBaseVNode("span", _hoisted_58$1, toDisplayString$1(importResult.value.failed), 1),
              createBaseVNode("span", _hoisted_59$1, toDisplayString$1(unref(t)("onboarding.phase3.statErr")), 1)
            ]),
            createBaseVNode("div", _hoisted_60$1, [
              createBaseVNode("span", _hoisted_61$1, toDisplayString$1(importResult.value.total), 1),
              createBaseVNode("span", _hoisted_62$1, toDisplayString$1(unref(t)("onboarding.phase3.statTotal")), 1)
            ])
          ])) : createCommentVNode("", true),
          ((_h = (_g = importResult.value) == null ? void 0 : _g.results) == null ? void 0 : _h.length) ? (openBlock(), createElementBlock("ul", _hoisted_63$1, [
            (openBlock(true), createElementBlock(Fragment, null, renderList(importResult.value.results, (r, i) => {
              var _a2, _b2;
              return openBlock(), createElementBlock("li", {
                key: i,
                class: normalizeClass(r.ok ? "result-ok" : "result-error")
              }, [
                createBaseVNode("span", _hoisted_64$1, toDisplayString$1(r.tool_id || r.tool || unref(t)("common.dash")), 1),
                createBaseVNode("span", _hoisted_65$1, [
                  createBaseVNode("code", null, toDisplayString$1(r.name || ((_b2 = (_a2 = r.canonical) == null ? void 0 : _a2.manifest) == null ? void 0 : _b2.name)), 1)
                ]),
                createBaseVNode("span", _hoisted_66$1, toDisplayString$1(r.error || r.message || (r.ok ? "OK" : "failed")), 1)
              ], 2);
            }), 128))
          ])) : createCommentVNode("", true),
          createBaseVNode("div", _hoisted_67$1, [
            createBaseVNode("button", {
              class: "ghost",
              onClick: reset
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:refresh",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("onboarding.phase3.btnAgain")), 1)
            ]),
            createBaseVNode("button", {
              class: "primary",
              onClick: goSkills
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:arrow-right",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("onboarding.phase3.btnGoSkills")), 1)
            ])
          ])
        ])) : createCommentVNode("", true)
      ]);
    };
  }
};
const OnboardingView = /* @__PURE__ */ _export_sfc(_sfc_main$7, [["__scopeId", "data-v-afbc7ae5"]]);
const _sfc_main$6 = {
  __name: "OnboardingImportDialog",
  props: {
    modelValue: { type: Boolean, default: false }
  },
  emits: ["update:modelValue", "imported"],
  setup(__props, { emit: __emit }) {
    const emit2 = __emit;
    const { t } = useI18n();
    const innerRef = /* @__PURE__ */ ref(null);
    function onDone(result) {
      emit2("imported", result);
    }
    return (_ctx, _cache) => {
      return openBlock(), createBlock(Modal, {
        "model-value": __props.modelValue,
        size: "full",
        title: unref(t)("onboarding.title"),
        "onUpdate:modelValue": _cache[0] || (_cache[0] = (v) => emit2("update:modelValue", v))
      }, {
        "title-icon": withCtx(() => [
          createVNode(unref(Icon), {
            icon: "mdi:tray-arrow-down",
            width: "18",
            height: "18"
          })
        ]),
        default: withCtx(() => [
          createVNode(OnboardingView, {
            ref_key: "innerRef",
            ref: innerRef,
            onDone
          }, null, 512)
        ]),
        _: 1
      }, 8, ["model-value", "title"]);
    };
  }
};
let _seq = 0;
const useToastStore = /* @__PURE__ */ defineStore("toast", {
  state: () => ({
    items: []
    // { id, type, message, duration, createdAt }
  }),
  actions: {
    // push 一条 toast;type: success | error | info
    push({ type = "info", message = "", duration } = {}) {
      if (!message) return;
      _seq += 1;
      const item = {
        id: _seq,
        type,
        message: String(message),
        duration: duration ?? (type === "error" ? 5e3 : 3e3),
        createdAt: Date.now()
      };
      this.items.push(item);
      while (this.items.length > 5) this.items.shift();
      setTimeout(() => this.dismiss(item.id), item.duration);
      return item.id;
    },
    success(message, duration) {
      return this.push({ type: "success", message, duration });
    },
    error(message, duration) {
      return this.push({ type: "error", message, duration });
    },
    info(message, duration) {
      return this.push({ type: "info", message, duration });
    },
    dismiss(id) {
      this.items = this.items.filter((x) => x.id !== id);
    },
    clear() {
      this.items = [];
    }
  }
});
const _hoisted_1$5 = { class: "skills-layout" };
const _hoisted_2$5 = { class: "skills-pane" };
const _hoisted_3$5 = { class: "left-topbar" };
const _hoisted_4$4 = ["title"];
const _hoisted_5$4 = ["title"];
const _hoisted_6$4 = { class: "left-search" };
const _hoisted_7$4 = ["placeholder", "title"];
const _hoisted_8$4 = {
  key: 0,
  class: "left-error"
};
const _hoisted_9$4 = ["aria-label"];
const _hoisted_10$4 = ["aria-selected", "onClick", "onKeyup"];
const _hoisted_11$4 = { class: "skill-item-main" };
const _hoisted_12$4 = { class: "skill-item-head" };
const _hoisted_13$4 = { class: "skill-item-name" };
const _hoisted_14$4 = { class: "skill-item-version" };
const _hoisted_15$4 = { class: "skill-item-meta" };
const _hoisted_16$4 = ["title"];
const _hoisted_17$4 = {
  key: 1,
  class: "skill-list-empty"
};
const _hoisted_18$4 = { class: "hint" };
const _hoisted_19$4 = {
  key: 2,
  class: "skill-list-loading"
};
const _hoisted_20$4 = {
  key: 3,
  class: "left-pager"
};
const _hoisted_21$4 = ["disabled"];
const _hoisted_22$4 = ["disabled"];
const _hoisted_23$4 = { class: "detail-pane" };
const _hoisted_24$4 = {
  key: 0,
  class: "detail-empty"
};
const _hoisted_25$4 = { class: "empty-title" };
const _hoisted_26$4 = { class: "detail-toolbar" };
const _hoisted_27$4 = { class: "detail-title-block" };
const _hoisted_28$4 = { class: "detail-title-row" };
const _hoisted_29$3 = { class: "detail-name" };
const _hoisted_30$3 = ["title"];
const _hoisted_31$3 = { class: "detail-title-actions" };
const _hoisted_32$3 = ["title"];
const _hoisted_33$3 = ["disabled"];
const _hoisted_34$3 = ["disabled"];
const _hoisted_35$3 = {
  key: 0,
  class: "spinner spinner-sm"
};
const _hoisted_36$3 = {
  key: 0,
  class: "detail-desc"
};
const _hoisted_37$3 = {
  key: 1,
  class: "detail-triggers-row"
};
const _hoisted_38$3 = { class: "triggers-label" };
const _hoisted_39$3 = { class: "meta-text" };
const _hoisted_40$3 = { class: "detail-actions" };
const _hoisted_41$2 = ["data-tip", "aria-label", "disabled"];
const _hoisted_42$2 = {
  key: 0,
  class: "spinner spinner-sm"
};
const _hoisted_43$2 = ["data-tip", "aria-label"];
const _hoisted_44$2 = ["data-tip", "aria-label"];
const _hoisted_45$2 = ["data-tip", "aria-label"];
const _hoisted_46$2 = ["data-tip", "aria-label"];
const _hoisted_47$2 = ["data-tip", "aria-label"];
const _hoisted_48$2 = {
  key: 0,
  class: "detail-section detail-edit-fields"
};
const _hoisted_49$2 = { class: "editor-field-full" };
const _hoisted_50$2 = ["placeholder", "disabled"];
const _hoisted_51$2 = { class: "editor-field-full" };
const _hoisted_52$2 = ["placeholder", "disabled"];
const _hoisted_53$2 = {
  key: 0,
  class: "message message-error"
};
const _hoisted_54$1 = {
  key: 1,
  class: "message message-error"
};
const _hoisted_55$1 = {
  key: 2,
  class: "detail-section"
};
const _hoisted_56 = { class: "section-header" };
const _hoisted_57 = {
  key: 0,
  class: "muted small-hint"
};
const _hoisted_58 = {
  key: 0,
  class: "section-loading"
};
const _hoisted_59 = {
  key: 1,
  class: "message message-error"
};
const _hoisted_60 = { class: "scope-row" };
const _hoisted_61 = { class: "scope-row-label" };
const _hoisted_62 = { class: "chip-row" };
const _hoisted_63 = ["title", "onClick"];
const _hoisted_64 = {
  key: 0,
  class: "spinner spinner-sm chip-spinner"
};
const _hoisted_65 = {
  key: 2,
  class: "chip-count"
};
const _hoisted_66 = {
  key: 0,
  class: "chip-tool-selected-hint muted"
};
const _hoisted_67 = {
  key: 0,
  class: "scope-row"
};
const _hoisted_68 = { class: "scope-row-label" };
const _hoisted_69 = { class: "chip-row" };
const _hoisted_70 = ["title", "onClick"];
const _hoisted_71 = {
  key: 0,
  class: "spinner spinner-sm chip-spinner"
};
const _hoisted_72 = {
  key: 2,
  class: "chip-mini-list"
};
const _hoisted_73 = {
  key: 0,
  class: "chip-empty muted"
};
const _hoisted_74 = { class: "detail-section detail-body" };
const _hoisted_75 = { class: "section-header" };
const _hoisted_76 = {
  key: 0,
  class: "message message-error"
};
const _hoisted_77 = ["placeholder"];
const _hoisted_78 = {
  key: 0,
  class: "detail-loading"
};
const _hoisted_79 = {
  key: 1,
  class: "message message-error"
};
const _hoisted_80 = ["innerHTML"];
const _hoisted_81 = {
  key: 3,
  class: "section-empty"
};
const _hoisted_82 = {
  key: 0,
  class: "message message-success"
};
const _hoisted_83 = {
  key: 1,
  class: "message message-error"
};
const _hoisted_84 = { class: "tag-create" };
const _hoisted_85 = ["placeholder"];
const _hoisted_86 = ["placeholder"];
const _hoisted_87 = ["disabled"];
const _hoisted_88 = {
  key: 2,
  class: "tag-actions"
};
const _hoisted_89 = { class: "diff-label" };
const _hoisted_90 = { value: 0 };
const _hoisted_91 = ["value"];
const _hoisted_92 = { value: 0 };
const _hoisted_93 = ["value"];
const _hoisted_94 = {
  key: 3,
  class: "tag-list"
};
const _hoisted_95 = { class: "tag-id" };
const _hoisted_96 = { class: "tag-name" };
const _hoisted_97 = { class: "tag-msg" };
const _hoisted_98 = { class: "tag-time" };
const _hoisted_99 = ["onClick"];
const _hoisted_100 = ["disabled", "onClick"];
const _hoisted_101 = ["onClick"];
const _hoisted_102 = {
  key: 4,
  class: "empty-state empty-state-sm"
};
const _hoisted_103 = { class: "empty-title" };
const _hoisted_104 = {
  key: 5,
  class: "diff-panel"
};
const _hoisted_105 = { class: "diff-header" };
const _hoisted_106 = { class: "diff-stats" };
const _hoisted_107 = { class: "stat stat-added" };
const _hoisted_108 = { class: "stat stat-removed" };
const _hoisted_109 = { class: "stat stat-modified" };
const _hoisted_110 = { class: "stat stat-unchanged" };
const _hoisted_111 = { class: "diff-file-header" };
const _hoisted_112 = { class: "diff-file-kind" };
const _hoisted_113 = { class: "diff-file-path" };
const _hoisted_114 = {
  key: 0,
  class: "diff-content"
};
const _hoisted_115 = { class: "diff-line-no" };
const _hoisted_116 = {
  key: 0,
  class: "test-status-badge"
};
const _hoisted_117 = {
  key: 1,
  class: "message message-error",
  style: { "margin": "0" }
};
const _hoisted_118 = {
  key: 2,
  class: "test-summary"
};
const _hoisted_119 = {
  key: 0,
  class: "test-list"
};
const _hoisted_120 = { class: "test-check-name" };
const _hoisted_121 = { class: "test-check-msg" };
const _hoisted_122 = {
  key: 1,
  class: "test-loading"
};
const _hoisted_123 = {
  key: 0,
  class: "editor-hint-bar"
};
const _hoisted_124 = { class: "editor-grid editor-grid-2" };
const _hoisted_125 = { class: "editor-field" };
const _hoisted_126 = ["placeholder", "disabled"];
const _hoisted_127 = { class: "editor-field" };
const _hoisted_128 = ["placeholder", "disabled"];
const _hoisted_129 = { class: "editor-field-full" };
const _hoisted_130 = { class: "scope-toggle-row" };
const _hoisted_131 = { class: "segmented" };
const _hoisted_132 = ["disabled"];
const _hoisted_133 = ["disabled"];
const _hoisted_134 = ["disabled"];
const _hoisted_135 = {
  value: 0,
  disabled: ""
};
const _hoisted_136 = ["value"];
const _hoisted_137 = { key: 0 };
const _hoisted_138 = {
  key: 1,
  class: "muted small-hint"
};
const _hoisted_139 = { class: "editor-field-full" };
const _hoisted_140 = { class: "chip-row apply-tools-row" };
const _hoisted_141 = ["title", "onClick"];
const _hoisted_142 = {
  key: 0,
  class: "chip-empty muted"
};
const _hoisted_143 = {
  key: 1,
  class: "chip-tool-selected-hint muted"
};
const _hoisted_144 = { class: "editor-field-full" };
const _hoisted_145 = { class: "editor-field-full" };
const _hoisted_146 = ["placeholder"];
const _hoisted_147 = { class: "editor-field-full" };
const _hoisted_148 = {
  key: 1,
  class: "message message-error",
  style: { "margin": "0 0 12px" }
};
const _hoisted_149 = { class: "confirm-message" };
const size$2 = 200;
const _sfc_main$5 = {
  __name: "SkillsView",
  setup(__props) {
    const { t } = useI18n();
    const keyword = /* @__PURE__ */ ref("");
    const loading = /* @__PURE__ */ ref(false);
    const error = /* @__PURE__ */ ref("");
    const items = /* @__PURE__ */ ref([]);
    const total = /* @__PURE__ */ ref(0);
    const page = /* @__PURE__ */ ref(1);
    const selectedKey = /* @__PURE__ */ ref(null);
    const current = /* @__PURE__ */ ref(null);
    const currentMd = /* @__PURE__ */ ref("");
    const currentBody = /* @__PURE__ */ ref("");
    const currentMeta = /* @__PURE__ */ reactive({ description: "", triggers: [] });
    const currentTagList = /* @__PURE__ */ ref([]);
    const currentLoading = /* @__PURE__ */ ref(false);
    const currentError = /* @__PURE__ */ ref("");
    const editing = /* @__PURE__ */ ref(false);
    const editBody = /* @__PURE__ */ ref("");
    const editDescription = /* @__PURE__ */ ref("");
    const editTriggersText = /* @__PURE__ */ ref("");
    const editError = /* @__PURE__ */ ref("");
    const editSaving = /* @__PURE__ */ ref(false);
    function startInlineEdit() {
      if (!current.value) return;
      editBody.value = currentBody.value || "";
      editDescription.value = currentMeta.description || "";
      editTriggersText.value = (currentMeta.triggers || []).join(", ");
      editError.value = "";
      editing.value = true;
    }
    function cancelInlineEdit() {
      editing.value = false;
      editBody.value = "";
      editDescription.value = "";
      editTriggersText.value = "";
      editError.value = "";
    }
    async function saveInlineEdit() {
      if (!current.value) return;
      editError.value = "";
      const newTriggers = (editTriggersText.value || "").split(/[,\n]/).map((s) => s.trim()).filter(Boolean);
      const newDescription = (editDescription.value || "").trim();
      editSaving.value = true;
      try {
        const targetSkill = { ...current.value };
        const existingApplies = scopeHits.value.filter((h2) => h2.exists).map((h2) => ({ tool_id: h2.tool_id, scope: h2.scope, project_id: h2.project_id || 0 }));
        currentMeta.description = newDescription;
        currentMeta.triggers = newTriggers;
        const newMd = rebuildSkillMd(editBody.value, newTriggers, newDescription);
        await updateSkill({
          scope: targetSkill.scope,
          project_id: targetSkill.project_id,
          name: targetSkill.name,
          version: targetSkill.version,
          source: targetSkill.source || "local",
          manifest: {
            name: targetSkill.name,
            version: targetSkill.version,
            description: newDescription,
            triggers: newTriggers
          },
          files: [{ path: "SKILL.md", content: newMd }]
        });
        currentMd.value = newMd;
        currentBody.value = extractBody(newMd);
        editing.value = false;
        if (existingApplies.length) {
          const failList = [];
          for (const a of existingApplies) {
            try {
              await applySkill({
                name: targetSkill.name,
                scope: a.scope,
                project_id: a.project_id,
                tools: [a.tool_id]
              });
            } catch (e) {
              failList.push({ tool: a.tool_id, scope: a.scope, project_id: a.project_id, msg: (e == null ? void 0 : e.message) || String(e) });
            }
          }
          if (failList.length) {
            toast.error(t("skills.editor.syncPartialFailed", {
              ok: existingApplies.length - failList.length,
              total: existingApplies.length
            }));
          } else {
            toast.success(t("skills.editor.syncAllSuccess", { n: existingApplies.length }));
          }
        } else {
          toast.info(t("skills.editor.syncNone", { name: targetSkill.name }));
        }
      } catch (e) {
        editError.value = (e == null ? void 0 : e.message) || String(e);
      } finally {
        editSaving.value = false;
      }
    }
    function rebuildSkillMd(newBody, newTriggers, newDescription) {
      var _a, _b;
      const fm = {
        name: ((_a = current.value) == null ? void 0 : _a.name) || "",
        version: ((_b = current.value) == null ? void 0 : _b.version) || "",
        description: newDescription !== void 0 ? newDescription : currentMeta.description || "",
        triggers: newTriggers !== void 0 ? newTriggers : currentMeta.triggers || []
      };
      const yaml = Object.entries(fm).map(([k, v]) => Array.isArray(v) ? `${k}: [${v.map((x) => JSON.stringify(x)).join(", ")}]` : `${k}: ${JSON.stringify(v)}`).join("\n");
      return `---
${yaml}
---

${newBody || ""}
`;
    }
    const scopeTools = /* @__PURE__ */ ref([]);
    const scopeProjects = /* @__PURE__ */ ref([]);
    const scopeHits = /* @__PURE__ */ ref([]);
    const scopeLoading = /* @__PURE__ */ ref(false);
    const scopeError = /* @__PURE__ */ ref("");
    const selectedToolID = /* @__PURE__ */ ref(null);
    const syncingToolID = /* @__PURE__ */ ref(null);
    const flashTargetKey = /* @__PURE__ */ ref(null);
    let _flashTimer = null;
    function flashTarget(key) {
      flashTargetKey.value = key;
      if (_flashTimer) clearTimeout(_flashTimer);
      _flashTimer = setTimeout(() => {
        flashTargetKey.value = null;
      }, 2e3);
    }
    const toast = useToastStore();
    const toolDisplay = computed(() => {
      const m = {};
      for (const t2 of scopeTools.value) m[t2.tool_id] = t2.display_name || t2.tool_id;
      return m;
    });
    const TOOL_ICON_MAP = {
      codex: "mdi:console",
      claude: "mdi:robot-outline",
      opencode: "mdi:code-tags",
      cursor: "mdi:cursor-default-click-outline",
      trae: "mdi:leaf"
    };
    function toolIcon(toolID) {
      return TOOL_ICON_MAP[toolID] || "mdi:puzzle-outline";
    }
    function toolShort(toolID) {
      if (!toolID) return "?";
      return toolID.charAt(0).toUpperCase() + toolID.slice(1);
    }
    const scopeTargets = computed(() => {
      const map = /* @__PURE__ */ new Map();
      for (const h2 of scopeHits.value) {
        const key = h2.scope === "global" ? "global" : `p:${h2.project_id}`;
        if (!map.has(key)) {
          map.set(key, {
            key,
            scope: h2.scope,
            project_id: h2.project_id || 0,
            project_label: h2.project_label || (h2.scope === "global" ? t("skills.list.scopeGlobalChip") : ""),
            hits: [],
            existsCount: 0
          });
        }
        const e = map.get(key);
        e.hits.push(h2);
        if (h2.exists) e.existsCount++;
      }
      const list = Array.from(map.values());
      list.sort((a, b) => {
        if (a.scope !== b.scope) return a.scope === "global" ? -1 : 1;
        return a.project_id - b.project_id;
      });
      return list;
    });
    const scopeToolSummary = computed(() => {
      const out = [];
      for (const t2 of scopeTools.value) {
        const hits = scopeHits.value.filter((h2) => h2.tool_id === t2.tool_id);
        const hitCount = hits.filter((h2) => h2.exists).length;
        out.push({
          tool_id: t2.tool_id,
          display: t2.display_name || t2.tool_id,
          icon: toolIcon(t2.tool_id),
          hitCount,
          hasHit: hitCount > 0
        });
      }
      return out;
    });
    function selectedToolHitExists(target) {
      if (!selectedToolID.value) return false;
      const h2 = target.hits.find((x) => x.tool_id === selectedToolID.value);
      return !!(h2 && h2.exists);
    }
    function selectedToolBusy(target) {
      if (!selectedToolID.value) return false;
      return target.hits.some((h2) => h2.tool_id === selectedToolID.value && isBusy(h2.tool_id, h2.scope, h2.project_id));
    }
    async function loadScopeStatus({ silent = false } = {}) {
      if (!current.value) return;
      if (!silent) scopeLoading.value = true;
      scopeError.value = "";
      try {
        const resp = await getSkillScopeStatus({
          name: current.value.name,
          version: current.value.version
        });
        scopeTools.value = (resp == null ? void 0 : resp.tools) || [];
        scopeProjects.value = (resp == null ? void 0 : resp.projects) || [];
        scopeHits.value = (resp == null ? void 0 : resp.hits) || [];
        if (selectedToolID.value && !scopeTools.value.some((t2) => t2.tool_id === selectedToolID.value)) {
          selectedToolID.value = null;
        }
      } catch (e) {
        scopeError.value = (e == null ? void 0 : e.message) || String(e);
        if (!silent) {
          scopeTools.value = [];
          scopeProjects.value = [];
          scopeHits.value = [];
        }
        selectedToolID.value = null;
      } finally {
        if (!silent) scopeLoading.value = false;
      }
    }
    const busyKey = /* @__PURE__ */ ref("");
    function busyKeyFor(toolID, scope, projectID) {
      return `${toolID}|${scope}|${projectID || 0}`;
    }
    function isBusy(toolID, scope, projectID) {
      return busyKey.value === busyKeyFor(toolID, scope, projectID);
    }
    async function handleToolChipClick(toolSummary) {
      if (selectedToolID.value === toolSummary.tool_id) {
        selectedToolID.value = null;
        return;
      }
      selectedToolID.value = toolSummary.tool_id;
      syncingToolID.value = toolSummary.tool_id;
      try {
        await loadScopeStatus({ silent: true });
      } finally {
        syncingToolID.value = null;
      }
    }
    async function handleScopeChipClick(target) {
      if (!current.value) return;
      if (!selectedToolID.value) {
        toast.info(t("skills.list.scopeSelectToolFirst"));
        return;
      }
      const targetTool = selectedToolID.value;
      const targetHit = target.hits.find((h2) => h2.tool_id === targetTool);
      const toolLabel = toolDisplay.value[targetTool] || targetTool;
      if (targetHit && targetHit.exists) {
        const ok2 = await openConfirm({
          title: t("skills.list.unapplyConfirmTitle"),
          message: t("skills.list.unapplyConfirmMessage", {
            name: current.value.name,
            tool: toolLabel,
            scope: target.project_label
          }),
          confirmText: t("common.delete"),
          variant: "danger"
        });
        if (!ok2) return;
        await doUnapplyOne(targetHit);
        return;
      }
      const fakeHit = targetHit || {
        tool_id: targetTool,
        scope: target.scope,
        project_id: target.project_id || 0,
        exists: false
      };
      const ok = await openConfirm({
        title: t("skills.list.applyConfirmTitle"),
        message: t("skills.list.applyConfirmMessage", {
          name: current.value.name,
          tool: toolLabel,
          scope: target.project_label
        }),
        confirmText: t("common.confirm")
      });
      if (!ok) return;
      await doApplyOne(fakeHit);
    }
    async function doApplyOne(h2) {
      busyKey.value = busyKeyFor(h2.tool_id, h2.scope, h2.project_id);
      const targetSkill = current.value ? { ...current.value } : null;
      try {
        await applySkill({
          name: targetSkill.name,
          scope: h2.scope,
          project_id: h2.project_id || 0,
          tools: [h2.tool_id]
        });
        await loadScopeStatus();
        patchAppliedTools(targetSkill, h2.tool_id, h2.scope, "add");
        const targetKey = h2.scope === "global" ? "global" : `p:${h2.project_id}`;
        flashTarget(targetKey);
        const toolLabel = toolDisplay.value[h2.tool_id] || h2.tool_id;
        toast.success(t("skills.list.applySuccess", {
          path: `${toolLabel} · ${h2.scope === "global" ? t("skills.list.scopeGlobalChip") : `#${h2.project_id}`}`
        }));
      } catch (e) {
        toast.error(t("skills.list.applyFailed", { msg: (e == null ? void 0 : e.message) || String(e) }));
        scopeError.value = t("skills.list.applyFailed", { msg: (e == null ? void 0 : e.message) || String(e) });
      } finally {
        busyKey.value = "";
      }
    }
    async function doUnapplyOne(h2) {
      var _a;
      busyKey.value = busyKeyFor(h2.tool_id, h2.scope, h2.project_id);
      const targetSkill = current.value ? { ...current.value } : null;
      try {
        const list = await listApplies({
          scope: h2.scope,
          name: targetSkill.name,
          tool: h2.tool_id,
          status: "applied",
          page: 1,
          size: 1
          // 找最近一条即可
        });
        const last = (_a = list == null ? void 0 : list.items) == null ? void 0 : _a[0];
        if (!last) {
          await forceUndoApply({
            scope: h2.scope,
            project_id: h2.project_id || 0,
            name: targetSkill.name,
            tool: h2.tool_id
          });
          await loadScopeStatus();
          patchAppliedTools(targetSkill, h2.tool_id, h2.scope, "remove");
          const targetKey2 = h2.scope === "global" ? "global" : `p:${h2.project_id}`;
          flashTarget(targetKey2);
          const toolLabel2 = toolDisplay.value[h2.tool_id] || h2.tool_id;
          toast.success(t("skills.list.unapplySuccess", {
            path: `${toolLabel2} · ${h2.scope === "global" ? t("skills.list.scopeGlobalChip") : `#${h2.project_id}`}`
          }));
          return;
        }
        await undoApply({ apply_id: last.id });
        await loadScopeStatus();
        patchAppliedTools(targetSkill, h2.tool_id, h2.scope, "remove");
        const targetKey = h2.scope === "global" ? "global" : `p:${h2.project_id}`;
        flashTarget(targetKey);
        const toolLabel = toolDisplay.value[h2.tool_id] || h2.tool_id;
        toast.success(t("skills.list.unapplySuccess", {
          path: `${toolLabel} · ${h2.scope === "global" ? t("skills.list.scopeGlobalChip") : `#${h2.project_id}`}`
        }));
      } catch (e) {
        toast.error(t("skills.list.unapplyFailed", { msg: (e == null ? void 0 : e.message) || String(e) }));
        scopeError.value = t("skills.list.unapplyFailed", { msg: (e == null ? void 0 : e.message) || String(e) });
      } finally {
        busyKey.value = "";
      }
    }
    const aiOpen = /* @__PURE__ */ ref(false);
    function toggleAI() {
      aiOpen.value = !aiOpen.value;
    }
    function skillKey(p2) {
      if (!p2) return "";
      return p2.name || "";
    }
    function patchAppliedTools(targetSkill, toolId, scope, op) {
      if (!targetSkill || !toolId || scope !== "global") return;
      const idx = items.value.findIndex((x) => skillKey(x) === skillKey(targetSkill));
      if (idx < 0) return;
      const cur = items.value[idx];
      const curSet = new Set(cur.applied_tools || []);
      if (op === "add") curSet.add(toolId);
      else if (op === "remove") curSet.delete(toolId);
      items.value.splice(idx, 1, { ...cur, applied_tools: Array.from(curSet) });
    }
    const currentSkillMd = computed(() => currentBody.value || "");
    function onAIApply(text) {
      var _a, _b, _c, _d;
      const m = text.match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/);
      currentBody.value = m ? m[1].trim() : text.trim();
      const fm = text.match(/^---\n([\s\S]*?)\n---/);
      if (fm) {
        try {
          const block = fm[1];
          const desc = (_b = (_a = block.match(/description:\s*(.+)/)) == null ? void 0 : _a[1]) == null ? void 0 : _b.replace(/^["']|["']$/g, "");
          const trg = (_d = (_c = block.match(/triggers:\s*\[([^\]]*)\]/)) == null ? void 0 : _c[1]) == null ? void 0 : _d.split(",").map((s) => s.trim().replace(/^["']|["']$/g, "")).filter(Boolean);
          if (desc) currentMeta.description = desc;
          if (trg) currentMeta.triggers = trg;
        } catch (_) {
        }
      }
    }
    const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size$2)));
    async function reload() {
      loading.value = true;
      error.value = "";
      try {
        const resp = await listSkills({
          keyword: keyword.value || void 0,
          page: page.value,
          size: size$2
        });
        items.value = (resp == null ? void 0 : resp.items) || [];
        total.value = (resp == null ? void 0 : resp.total) || 0;
      } catch (e) {
        error.value = (e == null ? void 0 : e.message) || String(e);
      } finally {
        loading.value = false;
      }
    }
    async function loadCurrent(row) {
      var _a, _b, _c;
      if (!row) return;
      currentLoading.value = true;
      currentError.value = "";
      selectedToolID.value = null;
      scopeHits.value = [];
      scopeTools.value = [];
      scopeProjects.value = [];
      scopeError.value = "";
      try {
        const full = await getSkill({
          scope: row.scope,
          project_id: row.project_id,
          name: row.name,
          version: row.version,
          full: true
        });
        const c = ((_a = full == null ? void 0 : full.canonical) == null ? void 0 : _a.manifest) || {};
        const files = ((_b = full == null ? void 0 : full.canonical) == null ? void 0 : _b.files) || [];
        const md = ((_c = files.find((f) => f.path === "SKILL.md")) == null ? void 0 : _c.content) || "";
        currentMd.value = md;
        currentBody.value = extractBody(md);
        currentMeta.description = c.description || "";
        currentMeta.triggers = c.triggers || [];
        current.value = { ...row, _full: full };
        try {
          const out = await listTags({ skill_id: row.id });
          currentTagList.value = (out == null ? void 0 : out.items) || [];
        } catch (_) {
          currentTagList.value = [];
        }
        await loadScopeStatus();
      } catch (e) {
        currentError.value = (e == null ? void 0 : e.message) || String(e);
        current.value = { ...row };
        currentMd.value = "";
        currentBody.value = "";
      } finally {
        currentLoading.value = "";
      }
    }
    function extractBody(skillmd) {
      const m = skillmd.match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/);
      return m ? m[1].trim() : skillmd;
    }
    function selectItem(row) {
      if (editing.value) cancelInlineEdit();
      selectedKey.value = skillKey(row);
      loadCurrent(row);
    }
    watch(selectedKey, (k) => {
      if (!k) return;
      const row = items.value.find((x) => skillKey(x) === k);
      if (row) loadCurrent(row);
    });
    function onSearchEnter() {
      page.value = 1;
      reload();
    }
    function gotoPage(p2) {
      if (p2 < 1 || p2 > totalPages.value) return;
      page.value = p2;
      reload();
    }
    const filteredItems = computed(() => {
      const kw = keyword.value.trim().toLowerCase();
      if (!kw) return items.value;
      return items.value.filter((x) => (x.name || "").toLowerCase().includes(kw) || (x.version || "").toLowerCase().includes(kw));
    });
    const renderedHtml = computed(() => renderMarkdown(currentBody.value));
    const tagOpen = /* @__PURE__ */ ref(false);
    const tagList = /* @__PURE__ */ ref([]);
    const tagLoading = /* @__PURE__ */ ref(false);
    const tagError = /* @__PURE__ */ ref("");
    const tagMessage = /* @__PURE__ */ ref("");
    const newTagName = /* @__PURE__ */ ref("");
    const newTagMessage = /* @__PURE__ */ ref("");
    const diffResult = /* @__PURE__ */ ref(null);
    const diffLeftTagID = /* @__PURE__ */ ref(0);
    const diffRightTagID = /* @__PURE__ */ ref(0);
    const rolling = /* @__PURE__ */ ref(false);
    async function openTagDialog() {
      if (!current.value) return;
      tagOpen.value = true;
      tagList.value = [];
      diffResult.value = null;
      newTagName.value = "";
      newTagMessage.value = "";
      await loadTagList();
    }
    async function loadTagList() {
      if (!current.value) return;
      tagLoading.value = true;
      tagError.value = "";
      try {
        const out = await listTags({ scope: current.value.scope, name: current.value.name });
        tagList.value = (out == null ? void 0 : out.items) || [];
        currentTagList.value = tagList.value;
      } catch (e) {
        tagError.value = (e == null ? void 0 : e.message) || String(e);
      } finally {
        tagLoading.value = false;
      }
    }
    async function doCreateTag() {
      if (!current.value) {
        tagError.value = t("skills.tag.selectFirst");
        return;
      }
      if (!newTagName.value.trim()) {
        tagError.value = t("skills.tag.emptyName");
        return;
      }
      tagLoading.value = true;
      tagError.value = "";
      try {
        await createTag({
          scope: current.value.scope,
          project_id: current.value.project_id,
          name: current.value.name,
          tag: newTagName.value.trim(),
          message: newTagMessage.value
        });
        newTagName.value = "";
        newTagMessage.value = "";
        tagMessage.value = t("skills.tag.msgCreated");
        await loadTagList();
      } catch (e) {
        tagError.value = (e == null ? void 0 : e.message) || String(e);
      } finally {
        tagLoading.value = false;
      }
    }
    async function doDeleteTag(tagID) {
      const ok = await openConfirm({
        title: t("common.delete"),
        message: t("skills.tag.confirmDelete", { id: tagID }),
        variant: "danger",
        confirmText: t("common.delete")
      });
      if (!ok) return;
      try {
        await deleteTag({ tag_id: tagID });
        tagMessage.value = t("skills.tag.msgDeleted", { id: tagID });
        await loadTagList();
      } catch (e) {
        tagError.value = (e == null ? void 0 : e.message) || String(e);
      }
    }
    async function doDiff(leftID, rightID) {
      if (!current.value) {
        tagError.value = t("skills.tag.selectFirst");
        return;
      }
      try {
        const out = await diffTag({ scope: current.value.scope, name: current.value.name, left_tag_id: leftID || 0, right_tag_id: rightID || 0 });
        diffResult.value = out;
        diffLeftTagID.value = leftID;
        diffRightTagID.value = rightID;
      } catch (e) {
        tagError.value = (e == null ? void 0 : e.message) || String(e);
      }
    }
    async function doRollback(tagID) {
      const ok = await openConfirm({
        title: t("skills.tag.rollbackTo"),
        message: t("skills.tag.confirmRollback", { id: tagID }),
        confirmText: t("skills.tag.rollbackTo"),
        variant: "danger"
      });
      if (!ok) return;
      rolling.value = true;
      tagError.value = "";
      try {
        const out = await rollbackTag({ tag_id: tagID });
        tagMessage.value = t("skills.tag.msgRolledBack", { pre: out.pre_rollback_tag, files: out.files_restored });
        diffResult.value = null;
        await reload();
        const row = items.value.find((x) => skillKey(x) === selectedKey.value);
        if (row) await loadCurrent(row);
        await loadTagList();
      } catch (e) {
        tagError.value = (e == null ? void 0 : e.message) || String(e);
      } finally {
        rolling.value = false;
      }
    }
    const testOpen = /* @__PURE__ */ ref(false);
    const testing = /* @__PURE__ */ ref(false);
    const testError = /* @__PURE__ */ ref("");
    const lastTest = /* @__PURE__ */ ref(null);
    async function triggerTest() {
      if (!current.value) return;
      const ok = await openConfirm({
        title: t("skills.test.title"),
        message: t("skills.test.confirmRun", { name: current.value.name, version: current.value.version }),
        confirmText: t("skills.list.btnTest")
      });
      if (!ok) return;
      testOpen.value = true;
      testing.value = true;
      testError.value = "";
      lastTest.value = null;
      try {
        const out = await runSkillTest({
          scope: current.value.scope,
          project_id: current.value.project_id,
          name: current.value.name,
          version: current.value.version,
          trigger: "manual"
        });
        lastTest.value = out;
      } catch (e) {
        testError.value = (e == null ? void 0 : e.message) || String(e);
      } finally {
        testing.value = false;
      }
    }
    const openError = /* @__PURE__ */ ref("");
    async function openInFolder() {
      var _a, _b, _c;
      if (!current.value) return;
      openError.value = "";
      try {
        const sp = ((_b = (_a = current.value._full) == null ? void 0 : _a.canonical) == null ? void 0 : _b.source_path) || ((_c = current.value._full) == null ? void 0 : _c.source_path) || "";
        if (!sp) {
          openError.value = "no source path";
          return;
        }
        const r = await platform.fs.reveal(sp);
        if (r && r.ok === false && r.fallbackUrl) {
          platform.platform.openExternal(r.fallbackUrl);
        }
      } catch (e) {
        openError.value = t("skills.list.openFailed", { msg: (e == null ? void 0 : e.message) || String(e) });
      }
    }
    async function copySourcePath() {
      var _a, _b, _c;
      if (!current.value) return;
      const sp = ((_b = (_a = current.value._full) == null ? void 0 : _a.canonical) == null ? void 0 : _b.source_path) || ((_c = current.value._full) == null ? void 0 : _c.source_path) || "";
      if (!sp) return;
      try {
        await platform.platform.setClipboardText(sp);
      } catch (_) {
        try {
          await navigator.clipboard.writeText(sp);
        } catch (_2) {
        }
      }
    }
    const editorOpen = /* @__PURE__ */ ref(false);
    const draft = /* @__PURE__ */ reactive({
      scope: "global",
      project_id: 0,
      name: "",
      version: "0.1.0",
      description: "",
      triggersText: "",
      body: "",
      applyTools: []
      // 2026-06-26:新建时勾选的"适用工具"列表
    });
    const editingKey = /* @__PURE__ */ ref(null);
    const editorProjects = /* @__PURE__ */ ref([]);
    const editorProjectsLoading = /* @__PURE__ */ ref(false);
    function startNew() {
      Object.assign(draft, {
        scope: "global",
        project_id: 0,
        name: "",
        version: "0.1.0",
        description: "",
        triggersText: "",
        body: "",
        applyTools: []
      });
      editingKey.value = null;
      error.value = "";
      editorOpen.value = true;
      loadEditorProjects();
    }
    async function loadEditorProjects(keyword2 = "") {
      editorProjectsLoading.value = true;
      try {
        const out = await listProjects({ keyword: keyword2 || void 0, page: 1, size: 200 });
        editorProjects.value = (out == null ? void 0 : out.items) || [];
        if (draft.project_id === 0 && editorProjects.value.length) {
          draft.project_id = editorProjects.value[0].id || 0;
        }
      } catch (_) {
        editorProjects.value = [];
      } finally {
        editorProjectsLoading.value = false;
      }
    }
    const APPLY_TOOLS = [
      { tool_id: "codex", display: "Codex" },
      { tool_id: "claude", display: "Claude" },
      { tool_id: "opencode", display: "OpenCode" },
      { tool_id: "cursor", display: "Cursor" },
      { tool_id: "trae", display: "Trae" }
    ];
    function toggleApplyTool(toolID) {
      const i = draft.applyTools.indexOf(toolID);
      if (i >= 0) draft.applyTools.splice(i, 1);
      else draft.applyTools.push(toolID);
    }
    function isApplyToolChecked(toolID) {
      return draft.applyTools.includes(toolID);
    }
    function buildSkillMd() {
      const triggers = draft.triggersText.split(/[\n,]/).map((s) => s.trim()).filter(Boolean);
      const m = {
        name: draft.name,
        version: draft.version,
        description: draft.description,
        triggers
      };
      const yaml = Object.entries(m).map(([k, v]) => Array.isArray(v) ? `${k}: [${v.map((x) => JSON.stringify(x)).join(", ")}]` : `${k}: ${JSON.stringify(v)}`).join("\n");
      return `---
${yaml}
---

${draft.body || ""}
`;
    }
    async function submit() {
      error.value = "";
      if (!draft.name.trim()) {
        error.value = t("skills.editor.errNameEmpty");
        return;
      }
      if (draft.description.trim().length < 10) {
        error.value = t("skills.editor.errDescShort");
        return;
      }
      const triggers = draft.triggersText.split(/[\n,]/).map((s) => s.trim()).filter(Boolean);
      if (triggers.length === 0) {
        error.value = t("skills.editor.errTriggersEmpty");
        return;
      }
      if (draft.scope === "project" && !draft.project_id) {
        error.value = t("skills.editor.errProjectRequired");
        return;
      }
      const payload = {
        scope: draft.scope,
        project_id: draft.project_id,
        name: draft.name,
        version: draft.version,
        source: "local",
        manifest: { name: draft.name, version: draft.version, description: draft.description, triggers },
        files: [{ path: "SKILL.md", content: buildSkillMd() }]
      };
      try {
        if (editingKey.value) await updateSkill(payload);
        else await createSkill(payload);
        if (draft.applyTools.length) {
          const failList = [];
          for (const tid of draft.applyTools) {
            try {
              await applySkill({
                name: draft.name,
                scope: draft.scope,
                project_id: draft.project_id || 0,
                tools: [tid]
              });
            } catch (e) {
              failList.push({ tool: tid, msg: (e == null ? void 0 : e.message) || String(e) });
            }
          }
          if (failList.length) {
            toast.error(t("skills.editor.applyPartialFailed", {
              ok: draft.applyTools.length - failList.length,
              total: draft.applyTools.length,
              fails: failList.map((f) => f.tool).join(", ")
            }));
          } else {
            toast.success(t("skills.editor.applyAllSuccess", { n: draft.applyTools.length }));
          }
        }
        editorOpen.value = false;
        await reload();
        const row = items.value.find((x) => x.name === payload.name && x.version === payload.version);
        if (row) selectItem(row);
      } catch (e) {
        error.value = (e == null ? void 0 : e.message) || String(e);
      }
    }
    async function removeCurrent() {
      if (!current.value) return;
      const row = current.value;
      const ok = await openConfirm({
        title: t("common.delete"),
        message: t("skills.list.confirmDelete", { name: row.name, version: row.version }),
        variant: "danger",
        confirmText: t("common.delete")
      });
      if (!ok) return;
      try {
        await deleteSkill({ scope: row.scope, project_id: row.project_id, name: row.name, version: row.version });
        if (editing.value) cancelInlineEdit();
        current.value = null;
        selectedKey.value = null;
        await reload();
      } catch (e) {
        error.value = (e == null ? void 0 : e.message) || String(e);
      }
    }
    const confirmOpen = /* @__PURE__ */ ref(false);
    const confirmOpts = /* @__PURE__ */ reactive({
      title: "",
      message: "",
      confirmText: "",
      cancelText: "",
      variant: "default",
      resolve: null
    });
    function openConfirm(opts) {
      confirmOpts.title = opts.title || t("common.confirm");
      confirmOpts.message = opts.message || "";
      confirmOpts.confirmText = opts.confirmText || t("common.confirm");
      confirmOpts.cancelText = opts.cancelText || t("common.cancel");
      confirmOpts.variant = opts.variant || "default";
      confirmOpen.value = true;
      return new Promise((resolve2) => {
        confirmOpts.resolve = resolve2;
      });
    }
    function resolveConfirm(ok) {
      if (confirmOpts.resolve) confirmOpts.resolve(ok);
      confirmOpen.value = false;
    }
    function goOnboarding() {
      importOpen.value = true;
    }
    const listRefs = /* @__PURE__ */ ref([]);
    const importOpen = /* @__PURE__ */ ref(false);
    function onImported() {
      reload();
    }
    onMounted(() => {
      reload();
    });
    return (_ctx, _cache) => {
      return openBlock(), createElementBlock("div", _hoisted_1$5, [
        createBaseVNode("aside", _hoisted_2$5, [
          createBaseVNode("div", _hoisted_3$5, [
            createBaseVNode("button", {
              class: "left-action",
              title: unref(t)("skills.list.btnNewSkillTitle"),
              onClick: startNew
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:plus",
                width: "16",
                height: "16"
              }),
              createBaseVNode("span", null, toDisplayString$1(unref(t)("skills.list.btnNewSkill")), 1)
            ], 8, _hoisted_4$4),
            createBaseVNode("button", {
              class: "left-action",
              title: unref(t)("skills.list.btnImportSkillTitle"),
              onClick: goOnboarding
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:tray-arrow-down",
                width: "16",
                height: "16"
              }),
              createBaseVNode("span", null, toDisplayString$1(unref(t)("skills.list.btnImportSkill")), 1)
            ], 8, _hoisted_5$4)
          ]),
          createBaseVNode("div", _hoisted_6$4, [
            createVNode(unref(Icon), {
              icon: "mdi:magnify",
              width: "14",
              height: "14",
              class: "search-icon"
            }),
            withDirectives(createBaseVNode("input", {
              "onUpdate:modelValue": _cache[0] || (_cache[0] = ($event) => keyword.value = $event),
              placeholder: unref(t)("skills.searchPlaceholder"),
              class: "search-input",
              title: unref(t)("skills.list.searchTitle"),
              onKeyup: withKeys(onSearchEnter, ["enter"])
            }, null, 40, _hoisted_7$4), [
              [vModelText, keyword.value]
            ])
          ]),
          error.value ? (openBlock(), createElementBlock("p", _hoisted_8$4, [
            createVNode(unref(Icon), {
              icon: "mdi:alert-circle-outline",
              width: "12",
              height: "12"
            }),
            createTextVNode(" " + toDisplayString$1(error.value), 1)
          ])) : createCommentVNode("", true),
          createBaseVNode("ul", {
            class: "skill-list",
            role: "listbox",
            "aria-label": unref(t)("skills.title")
          }, [
            (openBlock(true), createElementBlock(Fragment, null, renderList(filteredItems.value, (p2, i) => {
              return openBlock(), createElementBlock("li", {
                key: p2.name,
                ref_for: true,
                ref: (el) => {
                  if (el) listRefs.value[i] = el;
                },
                tabindex: "0",
                role: "option",
                "aria-selected": selectedKey.value === skillKey(p2),
                class: normalizeClass(["skill-item", { "skill-item-active": selectedKey.value === skillKey(p2) }]),
                onClick: ($event) => selectItem(p2),
                onKeyup: withKeys(($event) => selectItem(p2), ["enter"])
              }, [
                _cache[28] || (_cache[28] = createBaseVNode("span", { class: "skill-item-bar" }, null, -1)),
                createBaseVNode("div", _hoisted_11$4, [
                  createBaseVNode("div", _hoisted_12$4, [
                    createBaseVNode("span", _hoisted_13$4, toDisplayString$1(p2.name), 1),
                    createBaseVNode("span", _hoisted_14$4, "@" + toDisplayString$1(p2.version), 1)
                  ]),
                  createBaseVNode("div", _hoisted_15$4, [
                    (openBlock(true), createElementBlock(Fragment, null, renderList(p2.applied_tools || [], (tid) => {
                      return openBlock(), createElementBlock("span", {
                        key: tid,
                        class: "skill-item-tool-chip",
                        title: unref(t)("skills.list.appliedGlobal", { tool: toolDisplay.value[tid] || tid })
                      }, [
                        createVNode(unref(Icon), {
                          icon: toolIcon(tid),
                          width: "11",
                          height: "11"
                        }, null, 8, ["icon"]),
                        createBaseVNode("span", null, toDisplayString$1(toolShort(tid)), 1)
                      ], 8, _hoisted_16$4);
                    }), 128))
                  ])
                ])
              ], 42, _hoisted_10$4);
            }), 128))
          ], 8, _hoisted_9$4),
          !loading.value && !filteredItems.value.length ? (openBlock(), createElementBlock("div", _hoisted_17$4, [
            createVNode(unref(Icon), {
              icon: "mdi:book-open-variant",
              width: "28",
              height: "28"
            }),
            createBaseVNode("p", null, toDisplayString$1(unref(t)("skills.list.emptyTitle")), 1),
            createBaseVNode("p", _hoisted_18$4, toDisplayString$1(unref(t)("skills.list.emptyHint")), 1)
          ])) : createCommentVNode("", true),
          loading.value ? (openBlock(), createElementBlock("div", _hoisted_19$4, [
            _cache[29] || (_cache[29] = createBaseVNode("span", { class: "spinner" }, null, -1)),
            createBaseVNode("span", null, toDisplayString$1(unref(t)("common.processing")), 1)
          ])) : createCommentVNode("", true),
          totalPages.value > 1 ? (openBlock(), createElementBlock("footer", _hoisted_20$4, [
            createBaseVNode("button", {
              disabled: page.value <= 1,
              onClick: _cache[1] || (_cache[1] = ($event) => gotoPage(page.value - 1))
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:chevron-left",
                width: "12",
                height: "12"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("common.prev")), 1)
            ], 8, _hoisted_21$4),
            createBaseVNode("span", null, toDisplayString$1(page.value) + " / " + toDisplayString$1(totalPages.value), 1),
            createBaseVNode("button", {
              disabled: page.value >= totalPages.value,
              onClick: _cache[2] || (_cache[2] = ($event) => gotoPage(page.value + 1))
            }, [
              createTextVNode(toDisplayString$1(unref(t)("common.next")) + " ", 1),
              createVNode(unref(Icon), {
                icon: "mdi:chevron-right",
                width: "12",
                height: "12"
              })
            ], 8, _hoisted_22$4)
          ])) : createCommentVNode("", true)
        ]),
        createBaseVNode("section", _hoisted_23$4, [
          !current.value ? (openBlock(), createElementBlock("div", _hoisted_24$4, [
            createVNode(unref(Icon), {
              icon: "mdi:cursor-default-click-outline",
              width: "40",
              height: "40"
            }),
            createBaseVNode("p", _hoisted_25$4, toDisplayString$1(unref(t)("skills.list.selectToView")), 1)
          ])) : (openBlock(), createElementBlock(Fragment, { key: 1 }, [
            createBaseVNode("header", _hoisted_26$4, [
              createBaseVNode("div", _hoisted_27$4, [
                createBaseVNode("div", _hoisted_28$4, [
                  createBaseVNode("h1", _hoisted_29$3, toDisplayString$1(current.value.name), 1),
                  createBaseVNode("code", {
                    class: "detail-version",
                    role: "button",
                    tabindex: "0",
                    title: unref(t)("skills.tag.titlePrefix"),
                    onClick: openTagDialog,
                    onKeyup: withKeys(openTagDialog, ["enter"])
                  }, "@" + toDisplayString$1(current.value.version), 41, _hoisted_30$3),
                  createBaseVNode("span", {
                    class: normalizeClass(["badge", current.value.source === "market" ? "blue" : "gray"])
                  }, toDisplayString$1(current.value.source || "local"), 3),
                  createBaseVNode("div", _hoisted_31$3, [
                    !editing.value ? (openBlock(), createElementBlock("button", {
                      key: 0,
                      class: "ghost-link",
                      title: unref(t)("common.edit"),
                      onClick: startInlineEdit
                    }, [
                      createVNode(unref(Icon), {
                        icon: "mdi:pencil",
                        width: "12",
                        height: "12"
                      }),
                      createTextVNode(" " + toDisplayString$1(unref(t)("common.edit")), 1)
                    ], 8, _hoisted_32$3)) : (openBlock(), createElementBlock(Fragment, { key: 1 }, [
                      createBaseVNode("button", {
                        class: "title-action-btn title-action-cancel",
                        disabled: editSaving.value,
                        onClick: cancelInlineEdit
                      }, [
                        createVNode(unref(Icon), {
                          icon: "mdi:close",
                          width: "13",
                          height: "13"
                        }),
                        createTextVNode(" " + toDisplayString$1(unref(t)("common.cancel")), 1)
                      ], 8, _hoisted_33$3),
                      createBaseVNode("button", {
                        class: "title-action-btn title-action-save",
                        disabled: editSaving.value,
                        onClick: saveInlineEdit
                      }, [
                        editSaving.value ? (openBlock(), createElementBlock("span", _hoisted_35$3)) : (openBlock(), createBlock(unref(Icon), {
                          key: 1,
                          icon: "mdi:content-save",
                          width: "13",
                          height: "13"
                        })),
                        createTextVNode(" " + toDisplayString$1(editSaving.value ? unref(t)("common.processing") : unref(t)("common.save")), 1)
                      ], 8, _hoisted_34$3)
                    ], 64))
                  ])
                ]),
                !editing.value && currentMeta.description ? (openBlock(), createElementBlock("p", _hoisted_36$3, toDisplayString$1(currentMeta.description), 1)) : createCommentVNode("", true),
                !editing.value && (currentMeta.triggers || []).length ? (openBlock(), createElementBlock("div", _hoisted_37$3, [
                  createBaseVNode("span", _hoisted_38$3, toDisplayString$1(unref(t)("skills.editor.triggers")), 1),
                  createBaseVNode("span", _hoisted_39$3, toDisplayString$1((currentMeta.triggers || []).join("、")), 1)
                ])) : createCommentVNode("", true)
              ]),
              createBaseVNode("div", _hoisted_40$3, [
                createBaseVNode("button", {
                  class: "icon-btn",
                  "data-tip": unref(t)("skills.list.tooltipTest"),
                  "aria-label": unref(t)("skills.list.tooltipTest"),
                  disabled: testing.value,
                  onClick: triggerTest
                }, [
                  testing.value ? (openBlock(), createElementBlock("span", _hoisted_42$2)) : (openBlock(), createBlock(unref(Icon), {
                    key: 1,
                    icon: "mdi:test-tube",
                    width: "16",
                    height: "16"
                  }))
                ], 8, _hoisted_41$2),
                createBaseVNode("button", {
                  class: "icon-btn",
                  "data-tip": unref(t)("skills.list.tooltipTag"),
                  "aria-label": unref(t)("skills.list.tooltipTag"),
                  onClick: openTagDialog
                }, [
                  createVNode(unref(Icon), {
                    icon: "mdi:tag-outline",
                    width: "16",
                    height: "16"
                  })
                ], 8, _hoisted_43$2),
                createBaseVNode("button", {
                  class: "icon-btn",
                  "data-tip": unref(t)("skills.list.tooltipOpenFolder"),
                  "aria-label": unref(t)("skills.list.tooltipOpenFolder"),
                  onClick: openInFolder
                }, [
                  createVNode(unref(Icon), {
                    icon: "mdi:folder-outline",
                    width: "16",
                    height: "16"
                  })
                ], 8, _hoisted_44$2),
                createBaseVNode("button", {
                  class: "icon-btn",
                  "data-tip": unref(t)("skills.list.copyPath"),
                  "aria-label": unref(t)("skills.list.copyPath"),
                  onClick: copySourcePath
                }, [
                  createVNode(unref(Icon), {
                    icon: "mdi:content-copy",
                    width: "16",
                    height: "16"
                  })
                ], 8, _hoisted_45$2),
                createBaseVNode("button", {
                  class: "icon-btn",
                  "data-tip": unref(t)("skills.list.tooltipDelete"),
                  "aria-label": unref(t)("skills.list.tooltipDelete"),
                  onClick: removeCurrent
                }, [
                  createVNode(unref(Icon), {
                    icon: "mdi:delete",
                    width: "16",
                    height: "16"
                  })
                ], 8, _hoisted_46$2),
                createBaseVNode("button", {
                  class: "icon-btn ai-btn",
                  "data-tip": aiOpen.value ? unref(t)("skills.btnAiClose") : unref(t)("skills.btnAiOpen"),
                  "aria-label": aiOpen.value ? unref(t)("skills.btnAiClose") : unref(t)("skills.btnAiOpen"),
                  onClick: toggleAI
                }, [
                  createVNode(unref(Icon), {
                    icon: aiOpen.value ? "mdi:robot" : "mdi:robot-outline",
                    width: "16",
                    height: "16"
                  }, null, 8, ["icon"])
                ], 8, _hoisted_47$2)
              ])
            ]),
            editing.value ? (openBlock(), createElementBlock("section", _hoisted_48$2, [
              createBaseVNode("div", _hoisted_49$2, [
                createBaseVNode("label", null, [
                  createTextVNode(toDisplayString$1(unref(t)("skills.editor.description")) + " ", 1),
                  createBaseVNode("small", null, "(" + toDisplayString$1(unref(t)("skills.editor.descriptionHint")) + ")", 1)
                ]),
                withDirectives(createBaseVNode("textarea", {
                  "onUpdate:modelValue": _cache[3] || (_cache[3] = ($event) => editDescription.value = $event),
                  class: "desc-editor",
                  rows: "2",
                  spellcheck: "false",
                  placeholder: unref(t)("skills.editor.descriptionHint"),
                  disabled: editSaving.value
                }, null, 8, _hoisted_50$2), [
                  [vModelText, editDescription.value]
                ])
              ]),
              createBaseVNode("div", _hoisted_51$2, [
                createBaseVNode("label", null, [
                  createTextVNode(toDisplayString$1(unref(t)("skills.editor.triggers")) + " ", 1),
                  createBaseVNode("small", null, "(" + toDisplayString$1(unref(t)("skills.editor.triggersHint")) + ")", 1)
                ]),
                withDirectives(createBaseVNode("textarea", {
                  "onUpdate:modelValue": _cache[4] || (_cache[4] = ($event) => editTriggersText.value = $event),
                  class: "triggers-editor",
                  rows: "1",
                  spellcheck: "false",
                  placeholder: unref(t)("skills.editor.triggersHintPlaceholder"),
                  disabled: editSaving.value
                }, null, 8, _hoisted_52$2), [
                  [vModelText, editTriggersText.value]
                ])
              ]),
              editError.value ? (openBlock(), createElementBlock("p", _hoisted_53$2, [
                createVNode(unref(Icon), {
                  icon: "mdi:alert-circle-outline",
                  width: "12",
                  height: "12"
                }),
                createTextVNode(" " + toDisplayString$1(editError.value), 1)
              ])) : createCommentVNode("", true)
            ])) : createCommentVNode("", true),
            openError.value ? (openBlock(), createElementBlock("p", _hoisted_54$1, [
              createVNode(unref(Icon), {
                icon: "mdi:alert-circle-outline",
                width: "12",
                height: "12"
              }),
              createTextVNode(" " + toDisplayString$1(openError.value), 1)
            ])) : createCommentVNode("", true),
            !editing.value ? (openBlock(), createElementBlock("section", _hoisted_55$1, [
              createBaseVNode("header", _hoisted_56, [
                createBaseVNode("h3", null, [
                  createVNode(unref(Icon), {
                    icon: "mdi:earth",
                    width: "14",
                    height: "14"
                  }),
                  createTextVNode(" " + toDisplayString$1(unref(t)("skills.list.scopeLabel")), 1)
                ]),
                !scopeLoading.value && scopeHits.value.length ? (openBlock(), createElementBlock("span", _hoisted_57, toDisplayString$1(unref(t)("skills.list.scopeHitCount", { n: scopeHits.value.filter((h2) => h2.exists).length })), 1)) : createCommentVNode("", true)
              ]),
              scopeLoading.value ? (openBlock(), createElementBlock("p", _hoisted_58, [..._cache[30] || (_cache[30] = [
                createBaseVNode("span", { class: "spinner spinner-sm" }, null, -1),
                createBaseVNode("span", { class: "muted" }, "…", -1)
              ])])) : scopeError.value ? (openBlock(), createElementBlock("p", _hoisted_59, [
                createVNode(unref(Icon), {
                  icon: "mdi:alert-circle-outline",
                  width: "12",
                  height: "12"
                }),
                createTextVNode(" " + toDisplayString$1(scopeError.value), 1)
              ])) : (openBlock(), createElementBlock(Fragment, { key: 2 }, [
                createBaseVNode("div", _hoisted_60, [
                  createBaseVNode("span", _hoisted_61, toDisplayString$1(unref(t)("skills.list.scopeToolsRow")), 1),
                  createBaseVNode("div", _hoisted_62, [
                    (openBlock(true), createElementBlock(Fragment, null, renderList(scopeToolSummary.value, (t2) => {
                      return openBlock(), createElementBlock("button", {
                        key: t2.tool_id,
                        type: "button",
                        class: normalizeClass([
                          "chip",
                          "chip-tool",
                          t2.hasHit ? "chip-active" : "chip-muted",
                          selectedToolID.value === t2.tool_id ? "chip-tool-selected" : "",
                          syncingToolID.value === t2.tool_id ? "chip-tool-syncing" : ""
                        ]),
                        title: t2.hasHit ? `${t2.display}: ${t2.hitCount} 处生效` : `${t2.display}: 0 处生效`,
                        onClick: ($event) => handleToolChipClick(t2)
                      }, [
                        syncingToolID.value === t2.tool_id ? (openBlock(), createElementBlock("span", _hoisted_64)) : (openBlock(), createBlock(unref(Icon), {
                          key: 1,
                          icon: t2.icon,
                          width: "12",
                          height: "12"
                        }, null, 8, ["icon"])),
                        createBaseVNode("span", null, toDisplayString$1(toolShort(t2.tool_id)), 1),
                        t2.hitCount > 0 ? (openBlock(), createElementBlock("span", _hoisted_65, toDisplayString$1(t2.hitCount), 1)) : createCommentVNode("", true)
                      ], 10, _hoisted_63);
                    }), 128)),
                    selectedToolID.value ? (openBlock(), createElementBlock("span", _hoisted_66, toDisplayString$1(unref(t)("skills.list.scopeToolSelected", { tool: toolDisplay.value[selectedToolID.value] || selectedToolID.value })), 1)) : createCommentVNode("", true)
                  ])
                ]),
                selectedToolID.value ? (openBlock(), createElementBlock("div", _hoisted_67, [
                  createBaseVNode("span", _hoisted_68, toDisplayString$1(unref(t)("skills.list.scopeTargetsRow")), 1),
                  createBaseVNode("div", _hoisted_69, [
                    (openBlock(true), createElementBlock(Fragment, null, renderList(scopeTargets.value, (tg) => {
                      return openBlock(), createElementBlock("button", {
                        key: tg.key,
                        type: "button",
                        class: normalizeClass([
                          "chip",
                          "chip-scope-target",
                          selectedToolHitExists(tg) ? "chip-active" : "chip-muted",
                          !selectedToolID.value ? "chip-target-no-tool" : "",
                          selectedToolBusy(tg) ? "chip-busy" : "",
                          flashTargetKey.value === tg.key ? "chip-flash" : ""
                        ]),
                        title: selectedToolHitExists(tg) ? unref(t)("skills.list.unapplyConfirmTitle") : unref(t)("skills.list.applyConfirmTitle"),
                        onClick: ($event) => handleScopeChipClick(tg)
                      }, [
                        selectedToolBusy(tg) ? (openBlock(), createElementBlock("span", _hoisted_71)) : (openBlock(), createBlock(unref(Icon), {
                          key: 1,
                          icon: tg.scope === "global" ? "mdi:earth" : "mdi:folder-outline",
                          width: "12",
                          height: "12"
                        }, null, 8, ["icon"])),
                        createBaseVNode("span", null, toDisplayString$1(tg.project_label), 1),
                        selectedToolHitExists(tg) ? (openBlock(), createElementBlock("span", _hoisted_72, [
                          createVNode(unref(Icon), {
                            icon: toolIcon(selectedToolID.value),
                            width: "10",
                            height: "10",
                            class: "chip-mini-icon"
                          }, null, 8, ["icon"])
                        ])) : createCommentVNode("", true)
                      ], 10, _hoisted_70);
                    }), 128)),
                    !scopeTargets.value.length ? (openBlock(), createElementBlock("span", _hoisted_73, toDisplayString$1(unref(t)("skills.list.scopeEmpty")), 1)) : createCommentVNode("", true)
                  ])
                ])) : createCommentVNode("", true)
              ], 64))
            ])) : createCommentVNode("", true),
            createBaseVNode("section", _hoisted_74, [
              createBaseVNode("header", _hoisted_75, [
                createBaseVNode("h3", null, [
                  createVNode(unref(Icon), {
                    icon: editing.value ? "mdi:pencil-box-outline" : "mdi:text-box-outline",
                    width: "14",
                    height: "14"
                  }, null, 8, ["icon"]),
                  createTextVNode(" " + toDisplayString$1(editing.value ? unref(t)("skills.list.bodyEditing") : unref(t)("skills.list.bodyTitle")), 1)
                ])
              ]),
              editError.value ? (openBlock(), createElementBlock("p", _hoisted_76, [
                createVNode(unref(Icon), {
                  icon: "mdi:alert-circle-outline",
                  width: "12",
                  height: "12"
                }),
                createTextVNode(" " + toDisplayString$1(editError.value), 1)
              ])) : createCommentVNode("", true),
              editing.value ? withDirectives((openBlock(), createElementBlock("textarea", {
                key: 1,
                "onUpdate:modelValue": _cache[5] || (_cache[5] = ($event) => editBody.value = $event),
                class: "md-editor",
                spellcheck: "false",
                placeholder: unref(t)("skills.list.bodyEmpty")
              }, null, 8, _hoisted_77)), [
                [vModelText, editBody.value]
              ]) : (openBlock(), createElementBlock(Fragment, { key: 2 }, [
                currentLoading.value ? (openBlock(), createElementBlock("div", _hoisted_78, [
                  _cache[31] || (_cache[31] = createBaseVNode("span", { class: "spinner" }, null, -1)),
                  createBaseVNode("span", null, toDisplayString$1(unref(t)("common.processing")), 1)
                ])) : currentError.value ? (openBlock(), createElementBlock("p", _hoisted_79, [
                  createVNode(unref(Icon), {
                    icon: "mdi:alert-circle-outline",
                    width: "12",
                    height: "12"
                  }),
                  createTextVNode(" " + toDisplayString$1(currentError.value), 1)
                ])) : currentBody.value ? (openBlock(), createElementBlock("div", {
                  key: 2,
                  class: "md-body",
                  innerHTML: renderedHtml.value
                }, null, 8, _hoisted_80)) : (openBlock(), createElementBlock("p", _hoisted_81, toDisplayString$1(unref(t)("skills.list.bodyEmpty")), 1))
              ], 64))
            ])
          ], 64))
        ]),
        aiOpen.value ? (openBlock(), createBlock(AIPanel, {
          key: 0,
          "context-text": currentSkillMd.value,
          onApply: onAIApply
        }, null, 8, ["context-text"])) : createCommentVNode("", true),
        createVNode(Modal, {
          modelValue: tagOpen.value,
          "onUpdate:modelValue": _cache[12] || (_cache[12] = ($event) => tagOpen.value = $event),
          size: "xl",
          title: current.value ? unref(t)("skills.tag.titlePrefix") + " — " + current.value.name + "@" + current.value.version : unref(t)("skills.tag.titlePrefix")
        }, {
          "title-icon": withCtx(() => [
            createVNode(unref(Icon), {
              icon: "mdi:tag-outline",
              width: "18",
              height: "18"
            })
          ]),
          default: withCtx(() => [
            tagMessage.value ? (openBlock(), createElementBlock("p", _hoisted_82, [
              createVNode(unref(Icon), {
                icon: "mdi:check-circle-outline",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(tagMessage.value), 1)
            ])) : createCommentVNode("", true),
            tagError.value ? (openBlock(), createElementBlock("p", _hoisted_83, [
              createVNode(unref(Icon), {
                icon: "mdi:alert-circle-outline",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(tagError.value), 1)
            ])) : createCommentVNode("", true),
            createBaseVNode("div", _hoisted_84, [
              withDirectives(createBaseVNode("input", {
                "onUpdate:modelValue": _cache[6] || (_cache[6] = ($event) => newTagName.value = $event),
                placeholder: unref(t)("skills.tag.createPlaceholder"),
                class: "tag-input"
              }, null, 8, _hoisted_85), [
                [vModelText, newTagName.value]
              ]),
              withDirectives(createBaseVNode("input", {
                "onUpdate:modelValue": _cache[7] || (_cache[7] = ($event) => newTagMessage.value = $event),
                placeholder: unref(t)("skills.tag.msgPlaceholder"),
                class: "tag-input"
              }, null, 8, _hoisted_86), [
                [vModelText, newTagMessage.value]
              ]),
              createBaseVNode("button", {
                class: "primary",
                disabled: tagLoading.value,
                onClick: doCreateTag
              }, toDisplayString$1(tagLoading.value ? unref(t)("common.processing") : unref(t)("skills.tag.btnCreate")), 9, _hoisted_87)
            ]),
            tagList.value.length ? (openBlock(), createElementBlock("div", _hoisted_88, [
              createBaseVNode("span", _hoisted_89, toDisplayString$1(unref(t)("skills.tag.diff")) + ":", 1),
              withDirectives(createBaseVNode("select", {
                "onUpdate:modelValue": _cache[8] || (_cache[8] = ($event) => diffLeftTagID.value = $event)
              }, [
                createBaseVNode("option", _hoisted_90, toDisplayString$1(unref(t)("skills.tag.current")), 1),
                (openBlock(true), createElementBlock(Fragment, null, renderList(tagList.value, (tg) => {
                  return openBlock(), createElementBlock("option", {
                    key: tg.tag_id || tg.ID || tg.id,
                    value: tg.tag_id || tg.ID || tg.id
                  }, toDisplayString$1(tg.tag) + " (" + toDisplayString$1((tg.created_at || "").slice(0, 16)) + ")" + toDisplayString$1(tg.is_implicit ? unref(t)("skills.tag.implicit") : ""), 9, _hoisted_91);
                }), 128))
              ], 512), [
                [vModelSelect, diffLeftTagID.value]
              ]),
              createVNode(unref(Icon), {
                icon: "mdi:arrow-right",
                width: "14",
                height: "14",
                class: "diff-arrow"
              }),
              withDirectives(createBaseVNode("select", {
                "onUpdate:modelValue": _cache[9] || (_cache[9] = ($event) => diffRightTagID.value = $event)
              }, [
                createBaseVNode("option", _hoisted_92, toDisplayString$1(unref(t)("skills.tag.current")), 1),
                (openBlock(true), createElementBlock(Fragment, null, renderList(tagList.value, (tg) => {
                  return openBlock(), createElementBlock("option", {
                    key: tg.tag_id || tg.ID || tg.id,
                    value: tg.tag_id || tg.ID || tg.id
                  }, toDisplayString$1(tg.tag) + " (" + toDisplayString$1((tg.created_at || "").slice(0, 16)) + ")" + toDisplayString$1(tg.is_implicit ? unref(t)("skills.tag.implicit") : ""), 9, _hoisted_93);
                }), 128))
              ], 512), [
                [vModelSelect, diffRightTagID.value]
              ]),
              createBaseVNode("button", {
                onClick: _cache[10] || (_cache[10] = ($event) => doDiff(diffLeftTagID.value, diffRightTagID.value))
              }, toDisplayString$1(unref(t)("skills.tag.seeDiff")), 1),
              createBaseVNode("button", {
                onClick: _cache[11] || (_cache[11] = ($event) => doDiff(0, 0))
              }, toDisplayString$1(unref(t)("skills.tag.clear")), 1)
            ])) : createCommentVNode("", true),
            tagList.value.length ? (openBlock(), createElementBlock("ul", _hoisted_94, [
              (openBlock(true), createElementBlock(Fragment, null, renderList(tagList.value, (tg) => {
                return openBlock(), createElementBlock("li", {
                  key: tg.tag_id || tg.ID || tg.id,
                  class: normalizeClass({ "tag-implicit": tg.is_implicit })
                }, [
                  createBaseVNode("span", _hoisted_95, "#" + toDisplayString$1(tg.tag_id || tg.ID || tg.id), 1),
                  createBaseVNode("span", _hoisted_96, [
                    createBaseVNode("code", null, toDisplayString$1(tg.tag), 1)
                  ]),
                  createBaseVNode("span", _hoisted_97, toDisplayString$1(tg.message || unref(t)("common.dash")), 1),
                  createBaseVNode("span", _hoisted_98, toDisplayString$1((tg.created_at || "").slice(0, 19)), 1),
                  createBaseVNode("button", {
                    class: "link",
                    onClick: ($event) => doDiff(tg.tag_id || tg.ID || tg.id, 0)
                  }, toDisplayString$1(unref(t)("skills.tag.vsCurrent")), 9, _hoisted_99),
                  createBaseVNode("button", {
                    class: "link",
                    disabled: rolling.value,
                    onClick: ($event) => doRollback(tg.tag_id || tg.ID || tg.id)
                  }, toDisplayString$1(rolling.value ? unref(t)("skills.tag.rollingBack") : unref(t)("skills.tag.rollbackTo")), 9, _hoisted_100),
                  createBaseVNode("button", {
                    class: "link danger",
                    onClick: ($event) => doDeleteTag(tg.tag_id || tg.ID || tg.id)
                  }, toDisplayString$1(unref(t)("common.delete")), 9, _hoisted_101)
                ], 2);
              }), 128))
            ])) : !tagLoading.value ? (openBlock(), createElementBlock("div", _hoisted_102, [
              createVNode(unref(Icon), {
                icon: "mdi:tag-off-outline",
                width: "36",
                height: "36"
              }),
              createBaseVNode("p", _hoisted_103, toDisplayString$1(unref(t)("common.dash")), 1)
            ])) : createCommentVNode("", true),
            diffResult.value ? (openBlock(), createElementBlock("div", _hoisted_104, [
              createBaseVNode("header", _hoisted_105, [
                createBaseVNode("h4", null, toDisplayString$1(unref(t)("skills.tag.resultTitle")), 1),
                createBaseVNode("div", _hoisted_106, [
                  createBaseVNode("span", _hoisted_107, "+" + toDisplayString$1(unref(t)("skills.tag.added", { n: diffResult.value.added })), 1),
                  createBaseVNode("span", _hoisted_108, "-" + toDisplayString$1(unref(t)("skills.tag.removed", { n: diffResult.value.removed })), 1),
                  createBaseVNode("span", _hoisted_109, "~" + toDisplayString$1(unref(t)("skills.tag.modified", { n: diffResult.value.modified })), 1),
                  createBaseVNode("span", _hoisted_110, "=" + toDisplayString$1(unref(t)("skills.tag.unchanged", { n: diffResult.value.unchanged })), 1)
                ])
              ]),
              (openBlock(true), createElementBlock(Fragment, null, renderList(diffResult.value.files, (f) => {
                var _a;
                return openBlock(), createElementBlock("div", {
                  key: f.path,
                  class: normalizeClass(["diff-file", `diff-kind-${f.kind}`])
                }, [
                  createBaseVNode("div", _hoisted_111, [
                    createBaseVNode("span", _hoisted_112, toDisplayString$1(f.kind), 1),
                    createBaseVNode("code", _hoisted_113, toDisplayString$1(f.path), 1)
                  ]),
                  ((_a = f.lines) == null ? void 0 : _a.length) ? (openBlock(), createElementBlock("pre", _hoisted_114, [
                    (openBlock(true), createElementBlock(Fragment, null, renderList(f.lines, (l, i) => {
                      return openBlock(), createElementBlock("span", {
                        key: i,
                        class: normalizeClass(`diff-line diff-line-${l.kind}`)
                      }, [
                        createBaseVNode("span", _hoisted_115, toDisplayString$1(l.left_no || "") + "|" + toDisplayString$1(l.right_no || ""), 1),
                        createTextVNode(toDisplayString$1(l.text) + "\n", 1)
                      ], 2);
                    }), 128))
                  ])) : createCommentVNode("", true)
                ], 2);
              }), 128))
            ])) : createCommentVNode("", true)
          ]),
          _: 1
        }, 8, ["modelValue", "title"]),
        createVNode(Modal, {
          modelValue: testOpen.value,
          "onUpdate:modelValue": _cache[13] || (_cache[13] = ($event) => testOpen.value = $event),
          size: "lg",
          title: unref(t)("skills.test.title")
        }, {
          "title-icon": withCtx(() => [
            createVNode(unref(Icon), {
              icon: "mdi:test-tube",
              width: "18",
              height: "18"
            })
          ]),
          default: withCtx(() => {
            var _a, _b, _c, _d, _e, _f, _g, _h;
            return [
              createBaseVNode("div", {
                class: normalizeClass(["test-status-row", `test-status-${((_b = (_a = lastTest.value) == null ? void 0 : _a.run) == null ? void 0 : _b.status) || "errored"}`])
              }, [
                ((_c = lastTest.value) == null ? void 0 : _c.run) ? (openBlock(), createElementBlock("span", _hoisted_116, toDisplayString$1(lastTest.value.run.status), 1)) : createCommentVNode("", true),
                testError.value ? (openBlock(), createElementBlock("p", _hoisted_117, [
                  createVNode(unref(Icon), {
                    icon: "mdi:alert-circle-outline",
                    width: "14",
                    height: "14"
                  }),
                  createTextVNode(" " + toDisplayString$1(unref(t)("skills.test.errPrefix")) + " " + toDisplayString$1(testError.value), 1)
                ])) : ((_e = (_d = lastTest.value) == null ? void 0 : _d.run) == null ? void 0 : _e.summary) ? (openBlock(), createElementBlock("p", _hoisted_118, toDisplayString$1(lastTest.value.run.summary), 1)) : createCommentVNode("", true)
              ], 2),
              ((_g = (_f = lastTest.value) == null ? void 0 : _f.results) == null ? void 0 : _g.length) ? (openBlock(), createElementBlock("ul", _hoisted_119, [
                (openBlock(true), createElementBlock(Fragment, null, renderList(lastTest.value.results, (r) => {
                  return openBlock(), createElementBlock("li", {
                    key: r.id || r.ID,
                    class: normalizeClass(`test-check test-check-${r.status}`)
                  }, [
                    createBaseVNode("span", _hoisted_120, toDisplayString$1(r.check), 1),
                    createBaseVNode("span", {
                      class: normalizeClass(["test-check-status", `status-${r.status}`])
                    }, toDisplayString$1(r.status), 3),
                    createBaseVNode("span", _hoisted_121, toDisplayString$1(r.message), 1)
                  ], 2);
                }), 128))
              ])) : createCommentVNode("", true),
              (openBlock(true), createElementBlock(Fragment, null, renderList(((_h = lastTest.value) == null ? void 0 : _h.results) || [], (r) => {
                return openBlock(), createElementBlock("details", {
                  key: `d-${r.id || r.ID}`,
                  class: "test-detail"
                }, [
                  createBaseVNode("summary", null, toDisplayString$1(r.check) + " detail", 1),
                  createBaseVNode("pre", null, toDisplayString$1(r.detail), 1)
                ]);
              }), 128)),
              testing.value ? (openBlock(), createElementBlock("div", _hoisted_122, [
                _cache[32] || (_cache[32] = createBaseVNode("span", { class: "spinner" }, null, -1)),
                createBaseVNode("span", null, toDisplayString$1(unref(t)("common.processing")), 1)
              ])) : createCommentVNode("", true)
            ];
          }),
          _: 1
        }, 8, ["modelValue", "title"]),
        createVNode(Modal, {
          modelValue: editorOpen.value,
          "onUpdate:modelValue": _cache[23] || (_cache[23] = ($event) => editorOpen.value = $event),
          size: "xl",
          title: editingKey.value ? unref(t)("skills.editor.titleEdit") : unref(t)("skills.editor.titleNew")
        }, {
          "title-icon": withCtx(() => [
            createVNode(unref(Icon), {
              icon: editingKey.value ? "mdi:pencil" : "mdi:plus",
              width: "18",
              height: "18"
            }, null, 8, ["icon"])
          ]),
          footer: withCtx(() => [
            createBaseVNode("button", {
              type: "button",
              class: "ghost",
              onClick: _cache[22] || (_cache[22] = ($event) => editorOpen.value = false)
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:close",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("common.cancel")), 1)
            ]),
            createBaseVNode("button", {
              type: "button",
              class: "primary",
              onClick: submit
            }, [
              createVNode(unref(Icon), {
                icon: editingKey.value ? "mdi:content-save" : "mdi:plus",
                width: "14",
                height: "14"
              }, null, 8, ["icon"]),
              createTextVNode(" " + toDisplayString$1(editingKey.value ? unref(t)("common.save") : unref(t)("common.create")), 1)
            ])
          ]),
          default: withCtx(() => [
            createBaseVNode("form", {
              class: "editor-form",
              onSubmit: withModifiers(submit, ["prevent"])
            }, [
              editingKey.value ? (openBlock(), createElementBlock("div", _hoisted_123, [
                createBaseVNode("code", null, toDisplayString$1(editingKey.value.name) + "@" + toDisplayString$1(editingKey.value.version), 1)
              ])) : createCommentVNode("", true),
              createBaseVNode("div", _hoisted_124, [
                createBaseVNode("div", _hoisted_125, [
                  createBaseVNode("label", null, toDisplayString$1(unref(t)("skills.editor.name")), 1),
                  withDirectives(createBaseVNode("input", {
                    "onUpdate:modelValue": _cache[14] || (_cache[14] = ($event) => draft.name = $event),
                    placeholder: unref(t)("skills.editor.nameHint"),
                    disabled: !!editingKey.value
                  }, null, 8, _hoisted_126), [
                    [vModelText, draft.name]
                  ])
                ]),
                createBaseVNode("div", _hoisted_127, [
                  createBaseVNode("label", null, toDisplayString$1(unref(t)("skills.editor.version")), 1),
                  withDirectives(createBaseVNode("input", {
                    "onUpdate:modelValue": _cache[15] || (_cache[15] = ($event) => draft.version = $event),
                    placeholder: unref(t)("skills.editor.versionHint"),
                    disabled: !!editingKey.value
                  }, null, 8, _hoisted_128), [
                    [vModelText, draft.version]
                  ])
                ])
              ]),
              createBaseVNode("div", _hoisted_129, [
                createBaseVNode("label", null, toDisplayString$1(unref(t)("skills.editor.scope")), 1),
                createBaseVNode("div", _hoisted_130, [
                  createBaseVNode("div", _hoisted_131, [
                    createBaseVNode("button", {
                      type: "button",
                      class: normalizeClass(["seg-btn", draft.scope === "global" ? "seg-active" : ""]),
                      disabled: !!editingKey.value,
                      onClick: _cache[16] || (_cache[16] = ($event) => draft.scope = "global")
                    }, [
                      createVNode(unref(Icon), {
                        icon: "mdi:earth",
                        width: "13",
                        height: "13"
                      }),
                      createTextVNode(" " + toDisplayString$1(unref(t)("skills.editor.scopeGlobal")), 1)
                    ], 10, _hoisted_132),
                    createBaseVNode("button", {
                      type: "button",
                      class: normalizeClass(["seg-btn", draft.scope === "project" ? "seg-active" : ""]),
                      disabled: !!editingKey.value,
                      onClick: _cache[17] || (_cache[17] = ($event) => draft.scope = "project")
                    }, [
                      createVNode(unref(Icon), {
                        icon: "mdi:folder-outline",
                        width: "13",
                        height: "13"
                      }),
                      createTextVNode(" " + toDisplayString$1(unref(t)("skills.editor.scopeProject")), 1)
                    ], 10, _hoisted_133)
                  ]),
                  draft.scope === "project" ? withDirectives((openBlock(), createElementBlock("select", {
                    key: 0,
                    "onUpdate:modelValue": _cache[18] || (_cache[18] = ($event) => draft.project_id = $event),
                    class: "project-select",
                    disabled: !!editingKey.value
                  }, [
                    createBaseVNode("option", _hoisted_135, toDisplayString$1(unref(t)("skills.editor.projectPick")), 1),
                    (openBlock(true), createElementBlock(Fragment, null, renderList(editorProjects.value, (p2) => {
                      return openBlock(), createElementBlock("option", {
                        key: p2.id,
                        value: p2.id
                      }, [
                        createTextVNode(toDisplayString$1(p2.alias || p2.name), 1),
                        p2.alias && p2.name ? (openBlock(), createElementBlock("span", _hoisted_137, " · " + toDisplayString$1(p2.name), 1)) : createCommentVNode("", true)
                      ], 8, _hoisted_136);
                    }), 128))
                  ], 8, _hoisted_134)), [
                    [
                      vModelSelect,
                      draft.project_id,
                      void 0,
                      { number: true }
                    ]
                  ]) : (openBlock(), createElementBlock("span", _hoisted_138, toDisplayString$1(unref(t)("skills.editor.scopeGlobalHint")), 1))
                ])
              ]),
              createBaseVNode("div", _hoisted_139, [
                createBaseVNode("label", null, [
                  createTextVNode(toDisplayString$1(unref(t)("skills.editor.applyTools")) + " ", 1),
                  createBaseVNode("small", null, "(" + toDisplayString$1(unref(t)("skills.editor.applyToolsHint")) + ")", 1)
                ]),
                createBaseVNode("div", _hoisted_140, [
                  (openBlock(), createElementBlock(Fragment, null, renderList(APPLY_TOOLS, (tool) => {
                    return createBaseVNode("button", {
                      key: tool.tool_id,
                      type: "button",
                      class: normalizeClass(["chip", "chip-tool-pick", isApplyToolChecked(tool.tool_id) ? "chip-active" : ""]),
                      title: tool.display,
                      onClick: ($event) => toggleApplyTool(tool.tool_id)
                    }, [
                      createVNode(unref(Icon), {
                        icon: toolIcon(tool.tool_id),
                        width: "12",
                        height: "12"
                      }, null, 8, ["icon"]),
                      createBaseVNode("span", null, toDisplayString$1(tool.display), 1)
                    ], 10, _hoisted_141);
                  }), 64)),
                  !draft.applyTools.length ? (openBlock(), createElementBlock("span", _hoisted_142, toDisplayString$1(unref(t)("skills.editor.applyToolsNone")), 1)) : (openBlock(), createElementBlock("span", _hoisted_143, toDisplayString$1(unref(t)("skills.editor.applyToolsSelected", { n: draft.applyTools.length })), 1))
                ])
              ]),
              createBaseVNode("div", _hoisted_144, [
                createBaseVNode("label", null, [
                  createTextVNode(toDisplayString$1(unref(t)("skills.editor.description")) + " ", 1),
                  createBaseVNode("small", null, "(" + toDisplayString$1(unref(t)("skills.editor.descriptionHint")) + ")", 1)
                ]),
                withDirectives(createBaseVNode("textarea", {
                  "onUpdate:modelValue": _cache[19] || (_cache[19] = ($event) => draft.description = $event),
                  rows: "2"
                }, null, 512), [
                  [vModelText, draft.description]
                ])
              ]),
              createBaseVNode("div", _hoisted_145, [
                createBaseVNode("label", null, [
                  createTextVNode(toDisplayString$1(unref(t)("skills.editor.triggers")) + " ", 1),
                  createBaseVNode("small", null, "(" + toDisplayString$1(unref(t)("skills.editor.triggersHint")) + ")", 1)
                ]),
                withDirectives(createBaseVNode("textarea", {
                  "onUpdate:modelValue": _cache[20] || (_cache[20] = ($event) => draft.triggersText = $event),
                  rows: "1",
                  placeholder: unref(t)("skills.editor.triggersHintPlaceholder")
                }, null, 8, _hoisted_146), [
                  [vModelText, draft.triggersText]
                ])
              ]),
              createBaseVNode("div", _hoisted_147, [
                createBaseVNode("label", null, toDisplayString$1(unref(t)("skills.editor.body")), 1),
                withDirectives(createBaseVNode("textarea", {
                  "onUpdate:modelValue": _cache[21] || (_cache[21] = ($event) => draft.body = $event),
                  rows: "14",
                  class: "code"
                }, null, 512), [
                  [vModelText, draft.body]
                ])
              ]),
              error.value ? (openBlock(), createElementBlock("p", _hoisted_148, [
                createVNode(unref(Icon), {
                  icon: "mdi:alert-circle-outline",
                  width: "14",
                  height: "14"
                }),
                createTextVNode(" " + toDisplayString$1(error.value), 1)
              ])) : createCommentVNode("", true)
            ], 32)
          ]),
          _: 1
        }, 8, ["modelValue", "title"]),
        createVNode(Modal, {
          modelValue: confirmOpen.value,
          "onUpdate:modelValue": _cache[26] || (_cache[26] = ($event) => confirmOpen.value = $event),
          size: "sm",
          title: confirmOpts.title,
          "close-on-mask": false
        }, {
          footer: withCtx(() => [
            createBaseVNode("button", {
              type: "button",
              class: "ghost",
              onClick: _cache[24] || (_cache[24] = ($event) => resolveConfirm(false))
            }, toDisplayString$1(confirmOpts.cancelText), 1),
            createBaseVNode("button", {
              type: "button",
              class: normalizeClass(confirmOpts.variant === "danger" ? "danger" : "primary"),
              onClick: _cache[25] || (_cache[25] = ($event) => resolveConfirm(true))
            }, toDisplayString$1(confirmOpts.confirmText), 3)
          ]),
          default: withCtx(() => [
            createBaseVNode("p", _hoisted_149, toDisplayString$1(confirmOpts.message), 1)
          ]),
          _: 1
        }, 8, ["modelValue", "title"]),
        createVNode(_sfc_main$6, {
          modelValue: importOpen.value,
          "onUpdate:modelValue": _cache[27] || (_cache[27] = ($event) => importOpen.value = $event),
          onImported
        }, null, 8, ["modelValue"])
      ]);
    };
  }
};
const SkillsView = /* @__PURE__ */ _export_sfc(_sfc_main$5, [["__scopeId", "data-v-c5a7fda3"]]);
function listSources() {
  return http.get("/api/skillbox/market/sources");
}
function listMarketSkills(params = {}) {
  return http.get("/api/skillbox/market/skills", params);
}
function refreshSource(sourceId) {
  return http.post("/api/skillbox/market/refresh", { source_id: sourceId });
}
function installMarketSkill(payload) {
  return http.post("/api/skillbox/market/install", payload);
}
const _hoisted_1$4 = { class: "market" };
const _hoisted_2$4 = { class: "view-header" };
const _hoisted_3$4 = { class: "view-title" };
const _hoisted_4$3 = { class: "view-icon view-icon-orange" };
const _hoisted_5$3 = { class: "card" };
const _hoisted_6$3 = { class: "toolbar" };
const _hoisted_7$3 = { class: "toolbar-left" };
const _hoisted_8$3 = { class: "toolbar-label" };
const _hoisted_9$3 = { value: "global" };
const _hoisted_10$3 = {
  value: "project",
  disabled: ""
};
const _hoisted_11$3 = { class: "toolbar-center" };
const _hoisted_12$3 = { class: "search-box" };
const _hoisted_13$3 = ["placeholder"];
const _hoisted_14$3 = { class: "toolbar-right" };
const _hoisted_15$3 = ["disabled"];
const _hoisted_16$3 = {
  key: 0,
  class: "spinner"
};
const _hoisted_17$3 = { class: "source-tabs" };
const _hoisted_18$3 = ["onClick"];
const _hoisted_19$3 = { class: "source-type" };
const _hoisted_20$3 = {
  key: 0,
  class: "source-empty"
};
const _hoisted_21$3 = {
  key: 0,
  class: "message message-error"
};
const _hoisted_22$3 = {
  key: 1,
  class: "message message-success"
};
const _hoisted_23$3 = { class: "muted" };
const _hoisted_24$3 = {
  key: 2,
  class: "message message-success"
};
const _hoisted_25$3 = {
  key: 3,
  class: "message message-error"
};
const _hoisted_26$3 = { class: "table-container" };
const _hoisted_27$3 = {
  key: 0,
  class: "grid"
};
const _hoisted_28$3 = { class: "item-name" };
const _hoisted_29$2 = { class: "item-id" };
const _hoisted_30$2 = { class: "item-desc" };
const _hoisted_31$2 = { class: "row-actions" };
const _hoisted_32$2 = ["title", "onClick"];
const _hoisted_33$2 = ["disabled", "onClick"];
const _hoisted_34$2 = {
  key: 1,
  class: "empty-state"
};
const _hoisted_35$2 = { class: "empty-title" };
const _hoisted_36$2 = {
  key: 2,
  class: "loading-state"
};
const _hoisted_37$2 = {
  key: 4,
  class: "pager"
};
const _hoisted_38$2 = ["disabled"];
const _hoisted_39$2 = { class: "pager-info" };
const _hoisted_40$2 = ["disabled"];
const _hoisted_41$1 = {
  key: 0,
  class: "detail-grid"
};
const _hoisted_42$1 = { class: "detail-row" };
const _hoisted_43$1 = { class: "detail-label" };
const _hoisted_44$1 = { class: "detail-row" };
const _hoisted_45$1 = { class: "detail-label" };
const _hoisted_46$1 = { class: "detail-row" };
const _hoisted_47$1 = { class: "detail-id" };
const _hoisted_48$1 = { class: "detail-row detail-row-full" };
const _hoisted_49$1 = { class: "detail-label" };
const _hoisted_50$1 = { class: "detail-desc" };
const _hoisted_51$1 = {
  key: 0,
  class: "detail-row detail-row-full"
};
const _hoisted_52$1 = { class: "detail-label" };
const _hoisted_53$1 = { class: "detail-tags" };
const _hoisted_54 = ["disabled"];
const _hoisted_55 = { class: "confirm-message" };
const size$1 = 20;
const _sfc_main$4 = {
  __name: "MarketView",
  setup(__props) {
    const { t } = useI18n();
    const loading = /* @__PURE__ */ ref(false);
    const error = /* @__PURE__ */ ref("");
    const sources = /* @__PURE__ */ ref([]);
    const activeSourceId = /* @__PURE__ */ ref(0);
    const refreshing = /* @__PURE__ */ ref(false);
    const lastRefresh = /* @__PURE__ */ ref(null);
    const keyword = /* @__PURE__ */ ref("");
    const items = /* @__PURE__ */ ref([]);
    const total = /* @__PURE__ */ ref(0);
    const page = /* @__PURE__ */ ref(1);
    const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size$1)));
    const installScope = /* @__PURE__ */ ref("global");
    const installing = /* @__PURE__ */ ref(false);
    const installError = /* @__PURE__ */ ref("");
    const installOk = /* @__PURE__ */ ref("");
    async function fetchSources() {
      try {
        const res = await listSources();
        sources.value = res.items || [];
        if (sources.value.length > 0 && !activeSourceId.value) {
          activeSourceId.value = sources.value[0].id;
        }
      } catch (e) {
        error.value = t("market.errLoadSources", { msg: (e == null ? void 0 : e.message) || e });
      }
    }
    async function fetchSkills() {
      if (!activeSourceId.value) return;
      loading.value = true;
      error.value = "";
      try {
        const res = await listMarketSkills({
          source_id: activeSourceId.value,
          keyword: keyword.value,
          page: page.value,
          size: size$1
        });
        items.value = res.items || [];
        total.value = res.total || 0;
      } catch (e) {
        error.value = t("market.errLoadList", { msg: (e == null ? void 0 : e.message) || e });
      } finally {
        loading.value = false;
      }
    }
    async function onRefresh() {
      if (!activeSourceId.value || refreshing.value) return;
      refreshing.value = true;
      error.value = "";
      try {
        const res = await refreshSource(activeSourceId.value);
        lastRefresh.value = res;
        page.value = 1;
        await fetchSkills();
      } catch (e) {
        error.value = t("market.errRefresh", { msg: (e == null ? void 0 : e.message) || e });
      } finally {
        refreshing.value = false;
      }
    }
    function onSearch() {
      page.value = 1;
      fetchSkills();
    }
    function onSelectSource(id) {
      activeSourceId.value = id;
      page.value = 1;
      lastRefresh.value = null;
      fetchSkills();
    }
    async function onInstall(item) {
      var _a, _b;
      const ok = await openConfirm({
        title: t("market.btnInstall"),
        message: t("market.installConfirm", { name: item.name, scope: installScope.value }),
        confirmText: t("market.btnInstall")
      });
      if (!ok) return;
      installing.value = true;
      installError.value = "";
      installOk.value = "";
      try {
        const res = await installMarketSkill({
          source_id: activeSourceId.value,
          remote_id: item.remote_id,
          scope: installScope.value,
          project_id: 0
        });
        installOk.value = t("market.okInstalled", { name: ((_a = res == null ? void 0 : res.skill) == null ? void 0 : _a.name) || item.name, version: ((_b = res == null ? void 0 : res.skill) == null ? void 0 : _b.version) || "?" });
      } catch (e) {
        installError.value = t("market.errInstall", { msg: (e == null ? void 0 : e.message) || e });
      } finally {
        installing.value = false;
      }
    }
    const detailOpen = /* @__PURE__ */ ref(false);
    const detailItem = /* @__PURE__ */ ref(null);
    function openDetail(item) {
      detailItem.value = item;
      detailOpen.value = true;
    }
    const confirmOpen = /* @__PURE__ */ ref(false);
    const confirmOpts = /* @__PURE__ */ reactive({
      title: "",
      message: "",
      confirmText: "",
      cancelText: "",
      variant: "default",
      resolve: null
    });
    function openConfirm(opts) {
      confirmOpts.title = opts.title || t("common.confirm");
      confirmOpts.message = opts.message || "";
      confirmOpts.confirmText = opts.confirmText || t("common.confirm");
      confirmOpts.cancelText = opts.cancelText || t("common.cancel");
      confirmOpts.variant = opts.variant || "default";
      confirmOpen.value = true;
      return new Promise((resolve2) => {
        confirmOpts.resolve = resolve2;
      });
    }
    function resolveConfirm(ok) {
      if (confirmOpts.resolve) confirmOpts.resolve(ok);
      confirmOpen.value = false;
    }
    onMounted(async () => {
      await fetchSources();
      await fetchSkills();
    });
    return (_ctx, _cache) => {
      var _a;
      return openBlock(), createElementBlock("div", _hoisted_1$4, [
        createBaseVNode("header", _hoisted_2$4, [
          createBaseVNode("div", _hoisted_3$4, [
            createBaseVNode("div", _hoisted_4$3, [
              createVNode(unref(Icon), {
                icon: "mdi:cart-outline",
                width: "24",
                height: "24"
              })
            ]),
            createBaseVNode("div", null, [
              createBaseVNode("h1", null, toDisplayString$1(unref(t)("market.title")), 1),
              createBaseVNode("p", null, toDisplayString$1(unref(t)("market.subtitle")), 1)
            ])
          ])
        ]),
        createBaseVNode("div", _hoisted_5$3, [
          createBaseVNode("div", _hoisted_6$3, [
            createBaseVNode("div", _hoisted_7$3, [
              createBaseVNode("span", _hoisted_8$3, toDisplayString$1(unref(t)("market.scopeLabel")), 1),
              withDirectives(createBaseVNode("select", {
                "onUpdate:modelValue": _cache[0] || (_cache[0] = ($event) => installScope.value = $event),
                class: "scope-select"
              }, [
                createBaseVNode("option", _hoisted_9$3, toDisplayString$1(unref(t)("market.scopeGlobal")), 1),
                createBaseVNode("option", _hoisted_10$3, toDisplayString$1(unref(t)("market.scopeProject")), 1)
              ], 512), [
                [vModelSelect, installScope.value]
              ])
            ]),
            createBaseVNode("div", _hoisted_11$3, [
              createBaseVNode("div", _hoisted_12$3, [
                createVNode(unref(Icon), {
                  icon: "mdi:magnify",
                  width: "16",
                  height: "16",
                  class: "search-icon"
                }),
                withDirectives(createBaseVNode("input", {
                  "onUpdate:modelValue": _cache[1] || (_cache[1] = ($event) => keyword.value = $event),
                  type: "text",
                  placeholder: unref(t)("market.searchPlaceholder"),
                  class: "search-input",
                  onKeyup: withKeys(onSearch, ["enter"])
                }, null, 40, _hoisted_13$3), [
                  [vModelText, keyword.value]
                ])
              ]),
              createBaseVNode("button", {
                class: "ghost",
                onClick: onSearch
              }, [
                createVNode(unref(Icon), {
                  icon: "mdi:magnify",
                  width: "14",
                  height: "14"
                }),
                createTextVNode(" " + toDisplayString$1(unref(t)("common.search")), 1)
              ])
            ]),
            createBaseVNode("div", _hoisted_14$3, [
              createBaseVNode("button", {
                class: "primary",
                disabled: refreshing.value || !activeSourceId.value,
                onClick: onRefresh
              }, [
                refreshing.value ? (openBlock(), createElementBlock("span", _hoisted_16$3)) : (openBlock(), createBlock(unref(Icon), {
                  key: 1,
                  icon: "mdi:refresh",
                  width: "14",
                  height: "14"
                })),
                createTextVNode(" " + toDisplayString$1(refreshing.value ? unref(t)("market.refreshing") : unref(t)("market.btnRefresh")), 1)
              ], 8, _hoisted_15$3)
            ])
          ]),
          createBaseVNode("nav", _hoisted_17$3, [
            (openBlock(true), createElementBlock(Fragment, null, renderList(sources.value, (s) => {
              return openBlock(), createElementBlock("button", {
                key: s.id,
                class: normalizeClass(["source-tab", { active: s.id === activeSourceId.value }]),
                onClick: ($event) => onSelectSource(s.id)
              }, [
                createVNode(unref(Icon), {
                  icon: "mdi:radio-tower",
                  width: "14",
                  height: "14"
                }),
                createTextVNode(" " + toDisplayString$1(s.name) + " ", 1),
                createBaseVNode("span", _hoisted_19$3, toDisplayString$1(s.type), 1)
              ], 10, _hoisted_18$3);
            }), 128)),
            !sources.value.length && !loading.value ? (openBlock(), createElementBlock("span", _hoisted_20$3, toDisplayString$1(unref(t)("market.noSources")), 1)) : createCommentVNode("", true)
          ]),
          error.value ? (openBlock(), createElementBlock("div", _hoisted_21$3, [
            createVNode(unref(Icon), {
              icon: "mdi:alert-circle-outline",
              width: "14",
              height: "14"
            }),
            createTextVNode(" " + toDisplayString$1(error.value), 1)
          ])) : createCommentVNode("", true),
          lastRefresh.value ? (openBlock(), createElementBlock("div", _hoisted_22$3, [
            createVNode(unref(Icon), {
              icon: "mdi:check-circle-outline",
              width: "14",
              height: "14"
            }),
            createTextVNode(" " + toDisplayString$1(unref(t)("market.lastRefresh", { pulled: lastRefresh.value.pulled_count, inserted: lastRefresh.value.inserted, updated: lastRefresh.value.updated })) + " ", 1),
            createBaseVNode("span", _hoisted_23$3, "(" + toDisplayString$1(lastRefresh.value.finished_at) + ")", 1)
          ])) : createCommentVNode("", true),
          installOk.value ? (openBlock(), createElementBlock("div", _hoisted_24$3, [
            createVNode(unref(Icon), {
              icon: "mdi:check-circle-outline",
              width: "14",
              height: "14"
            }),
            createTextVNode(" " + toDisplayString$1(installOk.value), 1)
          ])) : createCommentVNode("", true),
          installError.value ? (openBlock(), createElementBlock("div", _hoisted_25$3, [
            createVNode(unref(Icon), {
              icon: "mdi:alert-circle-outline",
              width: "14",
              height: "14"
            }),
            createTextVNode(" " + toDisplayString$1(installError.value), 1)
          ])) : createCommentVNode("", true),
          createBaseVNode("div", _hoisted_26$3, [
            items.value.length > 0 ? (openBlock(), createElementBlock("table", _hoisted_27$3, [
              createBaseVNode("thead", null, [
                createBaseVNode("tr", null, [
                  createBaseVNode("th", null, toDisplayString$1(unref(t)("market.colName")), 1),
                  createBaseVNode("th", null, toDisplayString$1(unref(t)("market.colVersion")), 1),
                  createBaseVNode("th", null, toDisplayString$1(unref(t)("market.colAuthor")), 1),
                  createBaseVNode("th", null, toDisplayString$1(unref(t)("market.colDescription")), 1),
                  createBaseVNode("th", null, toDisplayString$1(unref(t)("market.colTags")), 1),
                  _cache[10] || (_cache[10] = createBaseVNode("th", { style: { "width": "160px" } }, null, -1))
                ])
              ]),
              createBaseVNode("tbody", null, [
                (openBlock(true), createElementBlock(Fragment, null, renderList(items.value, (it) => {
                  return openBlock(), createElementBlock("tr", {
                    key: it.remote_id
                  }, [
                    createBaseVNode("td", null, [
                      createBaseVNode("span", _hoisted_28$3, toDisplayString$1(it.name), 1),
                      createBaseVNode("span", _hoisted_29$2, toDisplayString$1(it.remote_id), 1)
                    ]),
                    createBaseVNode("td", null, [
                      createBaseVNode("code", null, toDisplayString$1(it.version || unref(t)("common.dash")), 1)
                    ]),
                    createBaseVNode("td", null, toDisplayString$1(it.author || unref(t)("common.dash")), 1),
                    createBaseVNode("td", _hoisted_30$2, toDisplayString$1(it.description || unref(t)("common.dash")), 1),
                    createBaseVNode("td", null, [
                      (openBlock(true), createElementBlock(Fragment, null, renderList((it.tags || "").split(",").filter(Boolean), (tg) => {
                        return openBlock(), createElementBlock("span", {
                          key: tg,
                          class: "tag"
                        }, toDisplayString$1(tg), 1);
                      }), 128))
                    ]),
                    createBaseVNode("td", null, [
                      createBaseVNode("div", _hoisted_31$2, [
                        createBaseVNode("button", {
                          class: "action-btn",
                          title: unref(t)("common.edit"),
                          onClick: ($event) => openDetail(it)
                        }, [
                          createVNode(unref(Icon), {
                            icon: "mdi:eye-outline",
                            width: "12",
                            height: "12"
                          })
                        ], 8, _hoisted_32$2),
                        createBaseVNode("button", {
                          class: "install-btn",
                          disabled: installing.value,
                          onClick: ($event) => onInstall(it)
                        }, [
                          createVNode(unref(Icon), {
                            icon: "mdi:download",
                            width: "12",
                            height: "12"
                          }),
                          createTextVNode(" " + toDisplayString$1(installing.value ? unref(t)("market.installing") : unref(t)("market.btnInstall")), 1)
                        ], 8, _hoisted_33$2)
                      ])
                    ])
                  ]);
                }), 128))
              ])
            ])) : !loading.value ? (openBlock(), createElementBlock("div", _hoisted_34$2, [
              createVNode(unref(Icon), {
                icon: "mdi:radio-tower",
                width: "48",
                height: "48"
              }),
              createBaseVNode("p", _hoisted_35$2, toDisplayString$1(unref(t)("market.emptyFirstTime")), 1)
            ])) : (openBlock(), createElementBlock("div", _hoisted_36$2, [
              _cache[11] || (_cache[11] = createBaseVNode("span", { class: "spinner" }, null, -1)),
              createBaseVNode("p", null, toDisplayString$1(unref(t)("market.loading")), 1)
            ]))
          ]),
          totalPages.value > 1 ? (openBlock(), createElementBlock("footer", _hoisted_37$2, [
            createBaseVNode("button", {
              disabled: page.value <= 1,
              onClick: _cache[2] || (_cache[2] = ($event) => {
                page.value--;
                fetchSkills();
              })
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:chevron-left",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("common.prev")), 1)
            ], 8, _hoisted_38$2),
            createBaseVNode("span", _hoisted_39$2, toDisplayString$1(unref(t)("common.pageOf", { page: page.value, total: totalPages.value, count: total.value })), 1),
            createBaseVNode("button", {
              disabled: page.value >= totalPages.value,
              onClick: _cache[3] || (_cache[3] = ($event) => {
                page.value++;
                fetchSkills();
              })
            }, [
              createTextVNode(toDisplayString$1(unref(t)("common.next")) + " ", 1),
              createVNode(unref(Icon), {
                icon: "mdi:chevron-right",
                width: "14",
                height: "14"
              })
            ], 8, _hoisted_40$2)
          ])) : createCommentVNode("", true)
        ]),
        createVNode(Modal, {
          modelValue: detailOpen.value,
          "onUpdate:modelValue": _cache[6] || (_cache[6] = ($event) => detailOpen.value = $event),
          size: "lg",
          title: ((_a = detailItem.value) == null ? void 0 : _a.name) || ""
        }, {
          "title-icon": withCtx(() => [
            createVNode(unref(Icon), {
              icon: "mdi:information-outline",
              width: "18",
              height: "18"
            })
          ]),
          footer: withCtx(() => [
            createBaseVNode("button", {
              type: "button",
              class: "ghost",
              onClick: _cache[4] || (_cache[4] = ($event) => detailOpen.value = false)
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:close",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("common.close")), 1)
            ]),
            createBaseVNode("button", {
              type: "button",
              class: "primary",
              disabled: installing.value,
              onClick: _cache[5] || (_cache[5] = ($event) => {
                detailOpen.value = false;
                onInstall(detailItem.value);
              })
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:download",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("market.btnInstall")), 1)
            ], 8, _hoisted_54)
          ]),
          default: withCtx(() => [
            detailItem.value ? (openBlock(), createElementBlock("div", _hoisted_41$1, [
              createBaseVNode("div", _hoisted_42$1, [
                createBaseVNode("span", _hoisted_43$1, toDisplayString$1(unref(t)("market.colVersion")), 1),
                createBaseVNode("code", null, toDisplayString$1(detailItem.value.version || unref(t)("common.dash")), 1)
              ]),
              createBaseVNode("div", _hoisted_44$1, [
                createBaseVNode("span", _hoisted_45$1, toDisplayString$1(unref(t)("market.colAuthor")), 1),
                createBaseVNode("span", null, toDisplayString$1(detailItem.value.author || unref(t)("common.dash")), 1)
              ]),
              createBaseVNode("div", _hoisted_46$1, [
                _cache[12] || (_cache[12] = createBaseVNode("span", { class: "detail-label" }, "ID", -1)),
                createBaseVNode("code", _hoisted_47$1, toDisplayString$1(detailItem.value.remote_id), 1)
              ]),
              createBaseVNode("div", _hoisted_48$1, [
                createBaseVNode("span", _hoisted_49$1, toDisplayString$1(unref(t)("market.colDescription")), 1),
                createBaseVNode("p", _hoisted_50$1, toDisplayString$1(detailItem.value.description || unref(t)("common.dash")), 1)
              ]),
              detailItem.value.tags ? (openBlock(), createElementBlock("div", _hoisted_51$1, [
                createBaseVNode("span", _hoisted_52$1, toDisplayString$1(unref(t)("market.colTags")), 1),
                createBaseVNode("div", _hoisted_53$1, [
                  (openBlock(true), createElementBlock(Fragment, null, renderList(String(detailItem.value.tags).split(",").filter(Boolean), (tg) => {
                    return openBlock(), createElementBlock("span", {
                      key: tg,
                      class: "tag"
                    }, toDisplayString$1(tg), 1);
                  }), 128))
                ])
              ])) : createCommentVNode("", true)
            ])) : createCommentVNode("", true)
          ]),
          _: 1
        }, 8, ["modelValue", "title"]),
        createVNode(Modal, {
          modelValue: confirmOpen.value,
          "onUpdate:modelValue": _cache[9] || (_cache[9] = ($event) => confirmOpen.value = $event),
          size: "sm",
          title: confirmOpts.title,
          "close-on-mask": false
        }, {
          footer: withCtx(() => [
            createBaseVNode("button", {
              type: "button",
              class: "ghost",
              onClick: _cache[7] || (_cache[7] = ($event) => resolveConfirm(false))
            }, toDisplayString$1(confirmOpts.cancelText), 1),
            createBaseVNode("button", {
              type: "button",
              class: normalizeClass(confirmOpts.variant === "danger" ? "danger" : "primary"),
              onClick: _cache[8] || (_cache[8] = ($event) => resolveConfirm(true))
            }, toDisplayString$1(confirmOpts.confirmText), 3)
          ]),
          default: withCtx(() => [
            createBaseVNode("p", _hoisted_55, toDisplayString$1(confirmOpts.message), 1)
          ]),
          _: 1
        }, 8, ["modelValue", "title"])
      ]);
    };
  }
};
const MarketView = /* @__PURE__ */ _export_sfc(_sfc_main$4, [["__scopeId", "data-v-ecadd803"]]);
function listAuditLogs(params = {}) {
  return http.get("/api/skillbox/audit/logs", params);
}
function getAuditStats() {
  return http.get("/api/skillbox/audit/stats");
}
const _hoisted_1$3 = { class: "audit" };
const _hoisted_2$3 = { class: "view-header" };
const _hoisted_3$3 = { class: "view-title" };
const _hoisted_4$2 = { class: "view-icon view-icon-amber" };
const _hoisted_5$2 = { class: "stats-row" };
const _hoisted_6$2 = { class: "stat-card stat-main" };
const _hoisted_7$2 = { class: "stat-label" };
const _hoisted_8$2 = { class: "stat-value" };
const _hoisted_9$2 = { class: "stat-card" };
const _hoisted_10$2 = { class: "stat-label" };
const _hoisted_11$2 = { class: "chips-container" };
const _hoisted_12$2 = {
  key: 0,
  class: "muted"
};
const _hoisted_13$2 = { class: "stat-card" };
const _hoisted_14$2 = { class: "stat-label" };
const _hoisted_15$2 = { class: "chips-container" };
const _hoisted_16$2 = {
  key: 0,
  class: "muted"
};
const _hoisted_17$2 = {
  key: 0,
  class: "card placeholder"
};
const _hoisted_18$2 = { class: "empty-state" };
const _hoisted_19$2 = { class: "empty-title" };
const _hoisted_20$2 = { class: "empty-desc" };
const _hoisted_21$2 = { class: "empty-desc" };
const _hoisted_22$2 = {
  key: 1,
  class: "card"
};
const _hoisted_23$2 = { class: "card-header" };
const _hoisted_24$2 = { class: "card-sub" };
const _hoisted_25$2 = { class: "filters" };
const _hoisted_26$2 = { class: "filter-group" };
const _hoisted_27$2 = { class: "filter-label" };
const _hoisted_28$2 = ["value"];
const _hoisted_29$1 = { class: "filter-group" };
const _hoisted_30$1 = { class: "filter-label" };
const _hoisted_31$1 = ["placeholder"];
const _hoisted_32$1 = { class: "filter-group" };
const _hoisted_33$1 = { class: "filter-label" };
const _hoisted_34$1 = ["placeholder"];
const _hoisted_35$1 = { class: "table-container" };
const _hoisted_36$1 = {
  key: 0,
  class: "grid"
};
const _hoisted_37$1 = { style: { "width": "70px" } };
const _hoisted_38$1 = { style: { "width": "160px" } };
const _hoisted_39$1 = { style: { "width": "120px" } };
const _hoisted_40$1 = { style: { "width": "140px" } };
const _hoisted_41 = { style: { "width": "180px" } };
const _hoisted_42 = { class: "td-id" };
const _hoisted_43 = { class: "td-time" };
const _hoisted_44 = { class: "target-code" };
const _hoisted_45 = { class: "td-payload" };
const _hoisted_46 = { class: "payload-details" };
const _hoisted_47 = { class: "payload-content" };
const _hoisted_48 = {
  key: 1,
  class: "empty-state"
};
const _hoisted_49 = { class: "empty-title" };
const _hoisted_50 = {
  key: 0,
  class: "pager"
};
const _hoisted_51 = ["disabled"];
const _hoisted_52 = { class: "pager-info" };
const _hoisted_53 = ["disabled"];
const size = 20;
const _sfc_main$3 = {
  __name: "AuditView",
  setup(__props) {
    const { t } = useI18n();
    const backendReady = /* @__PURE__ */ ref(false);
    const loading = /* @__PURE__ */ ref(false);
    const error = /* @__PURE__ */ ref("");
    const logs = /* @__PURE__ */ ref([]);
    const total = /* @__PURE__ */ ref(0);
    const page = /* @__PURE__ */ ref(1);
    const stats = /* @__PURE__ */ ref({ total: 0, by_action: {}, by_actor: {} });
    const filterAction = /* @__PURE__ */ ref("");
    const filterActor = /* @__PURE__ */ ref("");
    const filterTargetType = /* @__PURE__ */ ref("");
    const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)));
    const ACTION_OPTIONS = [
      "",
      "create",
      "update",
      "delete",
      "apply",
      "undo",
      "tag_create",
      "tag_delete",
      "tag_rollback",
      "test_run",
      "market_install",
      "onboarding_import",
      "project_create",
      "project_delete"
    ];
    async function loadStats() {
      try {
        const s = await getAuditStats();
        stats.value = s || { total: 0, by_action: {}, by_actor: {} };
        backendReady.value = true;
      } catch (e) {
        backendReady.value = false;
      }
    }
    async function loadLogs() {
      loading.value = true;
      error.value = "";
      try {
        const res = await listAuditLogs({
          page: page.value,
          size,
          action: filterAction.value || void 0,
          actor: filterActor.value || void 0,
          target_type: filterTargetType.value || void 0
        });
        logs.value = (res == null ? void 0 : res.items) || [];
        total.value = (res == null ? void 0 : res.total) || 0;
        backendReady.value = true;
      } catch (e) {
        error.value = (e == null ? void 0 : e.message) || String(e);
        logs.value = [];
        total.value = 0;
        backendReady.value = false;
      } finally {
        loading.value = false;
      }
    }
    function reload() {
      page.value = 1;
      loadLogs();
    }
    function gotoPage(p2) {
      if (p2 >= 1 && p2 <= totalPages.value) {
        page.value = p2;
        loadLogs();
      }
    }
    const actionColor = (a) => {
      if (!a) return "";
      if (a.startsWith("create") || a.startsWith("tag_create") || a === "market_install" || a === "onboarding_import" || a === "project_create") return "ok";
      if (a.startsWith("delete") || a === "project_delete" || a === "tag_delete") return "err";
      if (a.startsWith("undo") || a === "tag_rollback") return "warn";
      return "";
    };
    onMounted(async () => {
      await loadStats();
      await loadLogs();
    });
    return (_ctx, _cache) => {
      return openBlock(), createElementBlock("div", _hoisted_1$3, [
        createBaseVNode("header", _hoisted_2$3, [
          createBaseVNode("div", _hoisted_3$3, [
            createBaseVNode("div", _hoisted_4$2, [
              createVNode(unref(Icon), {
                icon: "mdi:script-text-outline",
                width: "24",
                height: "24"
              })
            ]),
            createBaseVNode("div", null, [
              createBaseVNode("h1", null, toDisplayString$1(unref(t)("audit.title")), 1),
              createBaseVNode("p", null, toDisplayString$1(unref(t)("audit.subtitle")), 1)
            ])
          ])
        ]),
        createBaseVNode("div", _hoisted_5$2, [
          createBaseVNode("div", _hoisted_6$2, [
            createBaseVNode("div", _hoisted_7$2, toDisplayString$1(unref(t)("audit.statTotal")), 1),
            createBaseVNode("div", _hoisted_8$2, toDisplayString$1(stats.value.total || 0), 1)
          ]),
          createBaseVNode("div", _hoisted_9$2, [
            createBaseVNode("div", _hoisted_10$2, toDisplayString$1(unref(t)("audit.statByAction")), 1),
            createBaseVNode("div", _hoisted_11$2, [
              (openBlock(true), createElementBlock(Fragment, null, renderList(stats.value.by_action || {}, (c, a) => {
                return openBlock(), createElementBlock("span", {
                  key: a,
                  class: "chip"
                }, [
                  createBaseVNode("code", null, toDisplayString$1(a), 1),
                  _cache[5] || (_cache[5] = createTextVNode()),
                  createBaseVNode("strong", null, "×" + toDisplayString$1(c), 1)
                ]);
              }), 128)),
              !Object.keys(stats.value.by_action || {}).length ? (openBlock(), createElementBlock("span", _hoisted_12$2, toDisplayString$1(unref(t)("common.dash")), 1)) : createCommentVNode("", true)
            ])
          ]),
          createBaseVNode("div", _hoisted_13$2, [
            createBaseVNode("div", _hoisted_14$2, toDisplayString$1(unref(t)("audit.statByActor")), 1),
            createBaseVNode("div", _hoisted_15$2, [
              (openBlock(true), createElementBlock(Fragment, null, renderList(stats.value.by_actor || {}, (c, a) => {
                return openBlock(), createElementBlock("span", {
                  key: a,
                  class: "chip"
                }, [
                  createBaseVNode("code", null, toDisplayString$1(a), 1),
                  _cache[6] || (_cache[6] = createTextVNode()),
                  createBaseVNode("strong", null, "×" + toDisplayString$1(c), 1)
                ]);
              }), 128)),
              !Object.keys(stats.value.by_actor || {}).length ? (openBlock(), createElementBlock("span", _hoisted_16$2, toDisplayString$1(unref(t)("common.dash")), 1)) : createCommentVNode("", true)
            ])
          ])
        ]),
        !backendReady.value ? (openBlock(), createElementBlock("div", _hoisted_17$2, [
          createBaseVNode("div", _hoisted_18$2, [
            createVNode(unref(Icon), {
              icon: "mdi:construction",
              width: "48",
              height: "48"
            }),
            createBaseVNode("h3", _hoisted_19$2, toDisplayString$1(unref(t)("audit.placeholderTitle")), 1),
            createBaseVNode("p", _hoisted_20$2, toDisplayString$1(unref(t)("audit.placeholderHint1")), 1),
            createBaseVNode("p", _hoisted_21$2, toDisplayString$1(unref(t)("audit.placeholderHint2")), 1)
          ])
        ])) : (openBlock(), createElementBlock("div", _hoisted_22$2, [
          createBaseVNode("header", _hoisted_23$2, [
            createBaseVNode("h3", null, [
              createVNode(unref(Icon), {
                icon: "mdi:format-list-bulleted",
                width: "16",
                height: "16"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("audit.listTitle")) + " ", 1),
              createBaseVNode("span", _hoisted_24$2, "— " + toDisplayString$1(unref(t)("common.totalCount", { count: total.value })), 1)
            ])
          ]),
          createBaseVNode("div", _hoisted_25$2, [
            createBaseVNode("div", _hoisted_26$2, [
              createBaseVNode("label", _hoisted_27$2, toDisplayString$1(unref(t)("audit.filterAction")), 1),
              withDirectives(createBaseVNode("select", {
                "onUpdate:modelValue": _cache[0] || (_cache[0] = ($event) => filterAction.value = $event),
                onChange: reload
              }, [
                (openBlock(), createElementBlock(Fragment, null, renderList(ACTION_OPTIONS, (a) => {
                  return createBaseVNode("option", {
                    key: a,
                    value: a
                  }, toDisplayString$1(a || unref(t)("common.all")), 9, _hoisted_28$2);
                }), 64))
              ], 544), [
                [vModelSelect, filterAction.value]
              ])
            ]),
            createBaseVNode("div", _hoisted_29$1, [
              createBaseVNode("label", _hoisted_30$1, toDisplayString$1(unref(t)("audit.filterActor")), 1),
              withDirectives(createBaseVNode("input", {
                "onUpdate:modelValue": _cache[1] || (_cache[1] = ($event) => filterActor.value = $event),
                placeholder: unref(t)("audit.actorPlaceholder"),
                onKeyup: withKeys(reload, ["enter"])
              }, null, 40, _hoisted_31$1), [
                [vModelText, filterActor.value]
              ])
            ]),
            createBaseVNode("div", _hoisted_32$1, [
              createBaseVNode("label", _hoisted_33$1, toDisplayString$1(unref(t)("audit.filterTargetType")), 1),
              withDirectives(createBaseVNode("input", {
                "onUpdate:modelValue": _cache[2] || (_cache[2] = ($event) => filterTargetType.value = $event),
                placeholder: unref(t)("audit.targetTypePlaceholder"),
                onKeyup: withKeys(reload, ["enter"])
              }, null, 40, _hoisted_34$1), [
                [vModelText, filterTargetType.value]
              ])
            ]),
            createBaseVNode("button", {
              class: "primary filter-btn",
              onClick: reload
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:magnify",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("common.applyFilter")), 1)
            ])
          ]),
          createBaseVNode("div", _hoisted_35$1, [
            logs.value.length ? (openBlock(), createElementBlock("table", _hoisted_36$1, [
              createBaseVNode("thead", null, [
                createBaseVNode("tr", null, [
                  createBaseVNode("th", _hoisted_37$1, toDisplayString$1(unref(t)("audit.colId")), 1),
                  createBaseVNode("th", _hoisted_38$1, toDisplayString$1(unref(t)("audit.colTime")), 1),
                  createBaseVNode("th", _hoisted_39$1, toDisplayString$1(unref(t)("audit.colActor")), 1),
                  createBaseVNode("th", _hoisted_40$1, toDisplayString$1(unref(t)("audit.colAction")), 1),
                  createBaseVNode("th", _hoisted_41, toDisplayString$1(unref(t)("audit.colTarget")), 1),
                  createBaseVNode("th", null, toDisplayString$1(unref(t)("audit.colPayload")), 1)
                ])
              ]),
              createBaseVNode("tbody", null, [
                (openBlock(true), createElementBlock(Fragment, null, renderList(logs.value, (log) => {
                  return openBlock(), createElementBlock("tr", {
                    key: log.ID || log.id
                  }, [
                    createBaseVNode("td", _hoisted_42, toDisplayString$1(log.ID || log.id), 1),
                    createBaseVNode("td", _hoisted_43, toDisplayString$1((log.CreatedAt || log.created_at || "").slice(0, 19)), 1),
                    createBaseVNode("td", null, [
                      createBaseVNode("code", null, toDisplayString$1(log.Actor || log.actor), 1)
                    ]),
                    createBaseVNode("td", null, [
                      createBaseVNode("span", {
                        class: normalizeClass(["action-badge", `action-${actionColor(log.Action || log.action)}`])
                      }, toDisplayString$1(log.Action || log.action), 3)
                    ]),
                    createBaseVNode("td", null, [
                      createBaseVNode("code", _hoisted_44, toDisplayString$1(log.TargetType || log.target_type) + "#" + toDisplayString$1(log.TargetID || log.target_id), 1)
                    ]),
                    createBaseVNode("td", _hoisted_45, [
                      createBaseVNode("details", _hoisted_46, [
                        createBaseVNode("summary", null, toDisplayString$1(unref(t)("audit.seeMore")), 1),
                        createBaseVNode("pre", _hoisted_47, toDisplayString$1(log.Payload || log.payload || unref(t)("common.dash")), 1)
                      ])
                    ])
                  ]);
                }), 128))
              ])
            ])) : !loading.value ? (openBlock(), createElementBlock("div", _hoisted_48, [
              createVNode(unref(Icon), {
                icon: "mdi:inbox-outline",
                width: "48",
                height: "48"
              }),
              createBaseVNode("p", _hoisted_49, toDisplayString$1(unref(t)("audit.empty")), 1)
            ])) : createCommentVNode("", true)
          ]),
          totalPages.value > 1 ? (openBlock(), createElementBlock("footer", _hoisted_50, [
            createBaseVNode("button", {
              disabled: page.value <= 1,
              onClick: _cache[3] || (_cache[3] = ($event) => gotoPage(page.value - 1))
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:chevron-left",
                width: "14",
                height: "14"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("common.prev")), 1)
            ], 8, _hoisted_51),
            createBaseVNode("span", _hoisted_52, toDisplayString$1(unref(t)("common.pageOf", { page: page.value, total: totalPages.value, count: total.value })), 1),
            createBaseVNode("button", {
              disabled: page.value >= totalPages.value,
              onClick: _cache[4] || (_cache[4] = ($event) => gotoPage(page.value + 1))
            }, [
              createTextVNode(toDisplayString$1(unref(t)("common.next")) + " ", 1),
              createVNode(unref(Icon), {
                icon: "mdi:chevron-right",
                width: "14",
                height: "14"
              })
            ], 8, _hoisted_53)
          ])) : createCommentVNode("", true)
        ]))
      ]);
    };
  }
};
const AuditView = /* @__PURE__ */ _export_sfc(_sfc_main$3, [["__scopeId", "data-v-ede600ed"]]);
const useAppStore = /* @__PURE__ */ defineStore("app", {
  state: () => ({
    runMode: "web",
    needAuth: true,
    appName: "",
    baseURL: ""
  }),
  getters: {
    isWeb: (s) => s.runMode === "web",
    isDesktop: (s) => s.runMode === "desktop",
    authEnabled: (s) => s.needAuth,
    // 给 UI 用的展示名
    deployLabel: (s) => s.runMode === "desktop" ? "桌面端" : "Web"
  },
  actions: {
    setRuntime(rt) {
      if (!rt) return;
      if (rt.runMode) this.runMode = rt.runMode;
      if (typeof rt.needAuth === "boolean") this.needAuth = rt.needAuth;
      if (typeof rt.appName === "string") this.appName = rt.appName;
    },
    setBaseURL(u) {
      this.baseURL = u || "";
    }
  }
});
const _hoisted_1$2 = { class: "settings-view" };
const _hoisted_2$2 = { class: "view-header" };
const _hoisted_3$2 = { class: "view-title" };
const _hoisted_4$1 = { class: "view-icon view-icon-gray" };
const _hoisted_5$1 = {
  key: 0,
  class: "card"
};
const _hoisted_6$1 = { class: "card-header" };
const _hoisted_7$1 = { class: "card-sub" };
const _hoisted_8$1 = {
  key: 0,
  class: "error-box"
};
const _hoisted_9$1 = {
  key: 1,
  class: "pref-list"
};
const _hoisted_10$1 = { class: "pref-item" };
const _hoisted_11$1 = { class: "pref-info" };
const _hoisted_12$1 = { class: "pref-label" };
const _hoisted_13$1 = { class: "pref-hint" };
const _hoisted_14$1 = { class: "toggle" };
const _hoisted_15$1 = ["checked"];
const _hoisted_16$1 = { class: "pref-item" };
const _hoisted_17$1 = { class: "pref-info" };
const _hoisted_18$1 = { class: "pref-label" };
const _hoisted_19$1 = { class: "pref-hint" };
const _hoisted_20$1 = { class: "toggle" };
const _hoisted_21$1 = ["checked"];
const _hoisted_22$1 = { class: "pref-item" };
const _hoisted_23$1 = { class: "pref-info" };
const _hoisted_24$1 = { class: "pref-label" };
const _hoisted_25$1 = { class: "pref-hint" };
const _hoisted_26$1 = { class: "toggle" };
const _hoisted_27$1 = ["checked"];
const _hoisted_28$1 = { class: "pref-item" };
const _hoisted_29 = { class: "pref-info" };
const _hoisted_30 = { class: "pref-label" };
const _hoisted_31 = { class: "pref-hint" };
const _hoisted_32 = ["value", "placeholder"];
const _hoisted_33 = { class: "pref-item pref-item-action" };
const _hoisted_34 = { class: "pref-info" };
const _hoisted_35 = { class: "pref-label" };
const _hoisted_36 = { class: "pref-hint" };
const _hoisted_37 = {
  key: 0,
  class: "hint-box"
};
const _hoisted_38 = {
  key: 1,
  class: "card"
};
const _hoisted_39 = { class: "empty-state" };
const _hoisted_40 = { class: "empty-title" };
const _sfc_main$2 = {
  __name: "SettingsView",
  setup(__props) {
    const { t } = useI18n();
    const desktopPrefs = /* @__PURE__ */ reactive({
      start_minimized: "false",
      notify_enabled: "true",
      shortcut_enabled: "true",
      global_hotkey: "Cmd+Shift+S"
    });
    const store = useAppStore();
    const { isDesktop: isDesktop2 } = storeToRefs(store);
    const prefsSupported = /* @__PURE__ */ ref(isDesktop2.value);
    const saveHint = /* @__PURE__ */ ref("");
    const notifyTest = /* @__PURE__ */ ref("");
    async function loadPrefs() {
      if (!isDesktop2.value) return;
      try {
        const snap = await platform.prefs.getAll();
        for (const k of Object.keys(desktopPrefs)) {
          if (snap[k] != null) desktopPrefs[k] = snap[k];
        }
      } catch (e) {
        prefsSupported.value = false;
      }
    }
    async function savePref(key, value) {
      if (!isDesktop2.value) return;
      try {
        await platform.prefs.set(key, String(value));
        saveHint.value = t("settings.saved");
        setTimeout(() => saveHint.value = "", 1500);
      } catch (e) {
        saveHint.value = t("settings.errSave", { msg: (e == null ? void 0 : e.message) || e });
      }
    }
    function onToggleStart(v) {
      desktopPrefs.start_minimized = v ? "true" : "false";
      savePref("desktop.start_minimized", desktopPrefs.start_minimized);
    }
    function onToggleNotify(v) {
      desktopPrefs.notify_enabled = v ? "true" : "false";
      savePref("desktop.notify_enabled", desktopPrefs.notify_enabled);
    }
    function onToggleShortcut(v) {
      desktopPrefs.shortcut_enabled = v ? "true" : "false";
      savePref("desktop.shortcut_enabled", desktopPrefs.shortcut_enabled);
    }
    function onHotkeyChange(e) {
      const v = (e.target.value || "").trim();
      desktopPrefs.global_hotkey = v;
      savePref("desktop.global_hotkey", v);
    }
    async function testNotify() {
      notifyTest.value = "";
      try {
        if (desktopPrefs.notify_enabled !== "true") {
          notifyTest.value = t("settings.notifyDisabled");
          return;
        }
        await platform.notify.show("", t("settings.testTitle"), t("settings.testBody"));
        notifyTest.value = t("settings.notifySent");
      } catch (e) {
        notifyTest.value = t("settings.errNotify", { msg: (e == null ? void 0 : e.message) || e });
      }
    }
    onMounted(loadPrefs);
    return (_ctx, _cache) => {
      return openBlock(), createElementBlock("div", _hoisted_1$2, [
        createBaseVNode("header", _hoisted_2$2, [
          createBaseVNode("div", _hoisted_3$2, [
            createBaseVNode("div", _hoisted_4$1, [
              createVNode(unref(Icon), {
                icon: "mdi:cog-outline",
                width: "24",
                height: "24"
              })
            ]),
            createBaseVNode("div", null, [
              createBaseVNode("h1", null, toDisplayString$1(unref(t)("settings.title")), 1),
              createBaseVNode("p", null, toDisplayString$1(unref(t)("settings.subtitle")), 1)
            ])
          ])
        ]),
        unref(isDesktop2) ? (openBlock(), createElementBlock("section", _hoisted_5$1, [
          createBaseVNode("header", _hoisted_6$1, [
            createBaseVNode("h3", null, [
              createVNode(unref(Icon), {
                icon: "mdi:desktop-classic",
                width: "18",
                height: "18"
              }),
              createTextVNode(" " + toDisplayString$1(unref(t)("settings.desktop.title")) + " ", 1),
              createBaseVNode("span", _hoisted_7$1, "— " + toDisplayString$1(unref(t)("settings.desktop.subtitle")), 1)
            ])
          ]),
          !prefsSupported.value ? (openBlock(), createElementBlock("div", _hoisted_8$1, [
            createVNode(unref(Icon), {
              icon: "mdi:alert-circle-outline",
              width: "16",
              height: "16"
            }),
            createTextVNode(" " + toDisplayString$1(unref(t)("settings.prefsUnavailable")), 1)
          ])) : (openBlock(), createElementBlock("div", _hoisted_9$1, [
            createBaseVNode("div", _hoisted_10$1, [
              createBaseVNode("div", _hoisted_11$1, [
                createBaseVNode("div", _hoisted_12$1, toDisplayString$1(unref(t)("settings.desktop.startMinimized")), 1),
                createBaseVNode("div", _hoisted_13$1, toDisplayString$1(unref(t)("settings.desktop.startMinimizedHint")), 1)
              ]),
              createBaseVNode("label", _hoisted_14$1, [
                createBaseVNode("input", {
                  type: "checkbox",
                  checked: desktopPrefs.start_minimized === "true",
                  onChange: _cache[0] || (_cache[0] = (e) => onToggleStart(e.target.checked))
                }, null, 40, _hoisted_15$1),
                _cache[3] || (_cache[3] = createBaseVNode("span", { class: "toggle-slider" }, null, -1))
              ])
            ]),
            createBaseVNode("div", _hoisted_16$1, [
              createBaseVNode("div", _hoisted_17$1, [
                createBaseVNode("div", _hoisted_18$1, toDisplayString$1(unref(t)("settings.desktop.notifyEnabled")), 1),
                createBaseVNode("div", _hoisted_19$1, toDisplayString$1(unref(t)("settings.desktop.notifyEnabledHint")), 1)
              ]),
              createBaseVNode("label", _hoisted_20$1, [
                createBaseVNode("input", {
                  type: "checkbox",
                  checked: desktopPrefs.notify_enabled === "true",
                  onChange: _cache[1] || (_cache[1] = (e) => onToggleNotify(e.target.checked))
                }, null, 40, _hoisted_21$1),
                _cache[4] || (_cache[4] = createBaseVNode("span", { class: "toggle-slider" }, null, -1))
              ])
            ]),
            createBaseVNode("div", _hoisted_22$1, [
              createBaseVNode("div", _hoisted_23$1, [
                createBaseVNode("div", _hoisted_24$1, toDisplayString$1(unref(t)("settings.desktop.shortcutEnabled")), 1),
                createBaseVNode("div", _hoisted_25$1, toDisplayString$1(unref(t)("settings.desktop.shortcutEnabledHint")), 1)
              ]),
              createBaseVNode("label", _hoisted_26$1, [
                createBaseVNode("input", {
                  type: "checkbox",
                  checked: desktopPrefs.shortcut_enabled === "true",
                  onChange: _cache[2] || (_cache[2] = (e) => onToggleShortcut(e.target.checked))
                }, null, 40, _hoisted_27$1),
                _cache[5] || (_cache[5] = createBaseVNode("span", { class: "toggle-slider" }, null, -1))
              ])
            ]),
            createBaseVNode("div", _hoisted_28$1, [
              createBaseVNode("div", _hoisted_29, [
                createBaseVNode("div", _hoisted_30, toDisplayString$1(unref(t)("settings.desktop.globalHotkey")), 1),
                createBaseVNode("div", _hoisted_31, toDisplayString$1(unref(t)("settings.desktop.globalHotkeyHint")), 1)
              ]),
              createBaseVNode("input", {
                class: "hotkey-input",
                type: "text",
                value: desktopPrefs.global_hotkey,
                onChange: onHotkeyChange,
                placeholder: unref(t)("settings.desktop.globalHotkeyPh")
              }, null, 40, _hoisted_32)
            ]),
            createBaseVNode("div", _hoisted_33, [
              createBaseVNode("div", _hoisted_34, [
                createBaseVNode("div", _hoisted_35, toDisplayString$1(unref(t)("settings.testNotify")), 1),
                createBaseVNode("div", _hoisted_36, toDisplayString$1(unref(t)("settings.testNotifyHint")), 1)
              ]),
              createBaseVNode("button", {
                class: "primary",
                onClick: testNotify
              }, [
                createVNode(unref(Icon), {
                  icon: "mdi:bell-ring-outline",
                  width: "14",
                  height: "14"
                }),
                createTextVNode(" " + toDisplayString$1(unref(t)("settings.btnTestNotify")), 1)
              ])
            ]),
            saveHint.value || notifyTest.value ? (openBlock(), createElementBlock("div", _hoisted_37, [
              saveHint.value ? (openBlock(), createBlock(unref(Icon), {
                key: 0,
                icon: "mdi:check-circle",
                width: "14",
                height: "14",
                class: "hint-icon hint-success"
              })) : createCommentVNode("", true),
              notifyTest.value ? (openBlock(), createBlock(unref(Icon), {
                key: 1,
                icon: "mdi:information",
                width: "14",
                height: "14",
                class: "hint-icon"
              })) : createCommentVNode("", true),
              createBaseVNode("span", null, toDisplayString$1(saveHint.value || notifyTest.value), 1)
            ])) : createCommentVNode("", true)
          ]))
        ])) : (openBlock(), createElementBlock("section", _hoisted_38, [
          createBaseVNode("div", _hoisted_39, [
            createVNode(unref(Icon), {
              icon: "mdi:monitor-dashboard",
              width: "48",
              height: "48"
            }),
            createBaseVNode("p", _hoisted_40, toDisplayString$1(unref(t)("settings.webOnlyHint")), 1)
          ])
        ]))
      ]);
    };
  }
};
const SettingsView = /* @__PURE__ */ _export_sfc(_sfc_main$2, [["__scopeId", "data-v-55def57c"]]);
const _hoisted_1$1 = {
  class: "toast-stack",
  "aria-live": "polite",
  "aria-atomic": "false"
};
const _hoisted_2$1 = { class: "toast-message" };
const _hoisted_3$1 = ["onClick"];
const _sfc_main$1 = {
  __name: "ToastContainer",
  setup(__props) {
    const toast = useToastStore();
    const ICON_MAP = {
      success: "mdi:check-circle-outline",
      error: "mdi:alert-circle-outline",
      info: "mdi:information-outline"
    };
    return (_ctx, _cache) => {
      return openBlock(), createElementBlock("div", _hoisted_1$1, [
        createVNode(TransitionGroup, { name: "toast" }, {
          default: withCtx(() => [
            (openBlock(true), createElementBlock(Fragment, null, renderList(unref(toast).items, (item) => {
              return openBlock(), createElementBlock("div", {
                key: item.id,
                class: normalizeClass(["toast-item", `toast-${item.type}`]),
                role: "status"
              }, [
                createVNode(unref(Icon), {
                  icon: ICON_MAP[item.type] || ICON_MAP.info,
                  width: "16",
                  height: "16",
                  class: "toast-icon"
                }, null, 8, ["icon"]),
                createBaseVNode("span", _hoisted_2$1, toDisplayString$1(item.message), 1),
                createBaseVNode("button", {
                  class: "toast-close",
                  "aria-label": "close",
                  onClick: ($event) => unref(toast).dismiss(item.id)
                }, [
                  createVNode(unref(Icon), {
                    icon: "mdi:close",
                    width: "12",
                    height: "12"
                  })
                ], 8, _hoisted_3$1)
              ], 2);
            }), 128))
          ]),
          _: 1
        })
      ]);
    };
  }
};
const ToastContainer = /* @__PURE__ */ _export_sfc(_sfc_main$1, [["__scopeId", "data-v-1050d243"]]);
const _hoisted_1 = { class: "sidebar-brand" };
const _hoisted_2 = { class: "brand-icon" };
const _hoisted_3 = { class: "brand-text" };
const _hoisted_4 = { class: "brand-name" };
const _hoisted_5 = ["aria-label"];
const _hoisted_6 = { class: "sidebar-nav flex-1" };
const _hoisted_7 = ["onClick"];
const _hoisted_8 = { class: "nav-icon" };
const _hoisted_9 = { class: "nav-content" };
const _hoisted_10 = { class: "nav-label" };
const _hoisted_11 = {
  key: 0,
  class: "nav-indicator"
};
const _hoisted_12 = { class: "sidebar-footer" };
const _hoisted_13 = { class: "status-text" };
const _hoisted_14 = ["title"];
const _hoisted_15 = ["title"];
const _hoisted_16 = ["title"];
const _hoisted_17 = { class: "main-content flex flex-col min-w-0" };
const _hoisted_18 = { class: "topbar" };
const _hoisted_19 = { class: "topbar-left" };
const _hoisted_20 = ["aria-label"];
const _hoisted_21 = { class: "breadcrumb" };
const _hoisted_22 = { class: "breadcrumb-brand" };
const _hoisted_23 = { class: "breadcrumb-current" };
const _hoisted_24 = { class: "topbar-right" };
const _hoisted_25 = { class: "stat-badge stat-badge-blue" };
const _hoisted_26 = { class: "stat-badge stat-badge-violet" };
const _hoisted_27 = { class: "stat-badge stat-badge-emerald" };
const _hoisted_28 = { class: "content-area" };
const MIN_SIDEBAR_WIDTH = 200;
const MAX_SIDEBAR_WIDTH = 420;
const _sfc_main = {
  __name: "App",
  setup(__props) {
    const { t } = useI18n();
    const tab = /* @__PURE__ */ ref("skills");
    const eventBus = /* @__PURE__ */ (() => {
      const listeners = /* @__PURE__ */ new Map();
      return {
        on(name, fn) {
          if (!listeners.has(name)) listeners.set(name, /* @__PURE__ */ new Set());
          listeners.get(name).add(fn);
        },
        off(name, fn) {
          var _a;
          (_a = listeners.get(name)) == null ? void 0 : _a.delete(fn);
        },
        emit(name, payload) {
          var _a;
          (_a = listeners.get(name)) == null ? void 0 : _a.forEach((fn) => {
            try {
              fn(payload);
            } catch (e) {
              console.error(`[eventBus] ${name} listener error:`, e);
            }
          });
        }
      };
    })();
    provide("appBus", eventBus);
    const isDark = /* @__PURE__ */ ref(false);
    const sidebarWidth = /* @__PURE__ */ ref(MIN_SIDEBAR_WIDTH);
    onMounted(() => {
      const savedTheme = localStorage.getItem("theme");
      if (savedTheme === "dark") {
        isDark.value = true;
        document.documentElement.classList.add("dark");
      } else if (savedTheme === "light") {
        isDark.value = false;
        document.documentElement.classList.remove("dark");
      } else {
        const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
        isDark.value = prefersDark;
        if (prefersDark) {
          document.documentElement.classList.add("dark");
        }
      }
      const savedWidth = localStorage.getItem("sidebarWidth");
      if (savedWidth) {
        const w = parseInt(savedWidth, 10);
        if (w >= MIN_SIDEBAR_WIDTH && w <= MAX_SIDEBAR_WIDTH) {
          sidebarWidth.value = w;
        }
      }
    });
    function toggleTheme() {
      isDark.value = !isDark.value;
      if (isDark.value) {
        document.documentElement.classList.add("dark");
        localStorage.setItem("theme", "dark");
      } else {
        document.documentElement.classList.remove("dark");
        localStorage.setItem("theme", "light");
      }
    }
    const sidebarOpen = /* @__PURE__ */ ref(true);
    const isMobile = /* @__PURE__ */ ref(false);
    function checkViewport() {
      isMobile.value = window.innerWidth < 768;
      if (isMobile.value) sidebarOpen.value = false;
      else sidebarOpen.value = true;
    }
    onMounted(() => {
      checkViewport();
      window.addEventListener("resize", checkViewport);
    });
    onUnmounted(() => window.removeEventListener("resize", checkViewport));
    const isResizing = /* @__PURE__ */ ref(false);
    function startResize(e) {
      if (isMobile.value) return;
      isResizing.value = true;
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
      e.preventDefault();
    }
    function onResizeMove(e) {
      if (!isResizing.value) return;
      const newWidth = Math.min(MAX_SIDEBAR_WIDTH, Math.max(MIN_SIDEBAR_WIDTH, e.clientX));
      sidebarWidth.value = newWidth;
    }
    function stopResize() {
      if (!isResizing.value) return;
      isResizing.value = false;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
      localStorage.setItem("sidebarWidth", String(sidebarWidth.value));
    }
    onMounted(() => {
      window.addEventListener("mousemove", onResizeMove);
      window.addEventListener("mouseup", stopResize);
    });
    onUnmounted(() => {
      window.removeEventListener("mousemove", onResizeMove);
      window.removeEventListener("mouseup", stopResize);
    });
    const stats = /* @__PURE__ */ ref({
      skills: 0,
      projects: 0,
      toolsReady: 0,
      toolsTotal: 0
    });
    const backendOK = /* @__PURE__ */ ref(false);
    async function refreshStats() {
      try {
        const [skillRes, projRes, obRes] = await Promise.all([
          listSkills({ page: 1, size: 1 }).catch(() => ({ total: 0 })),
          listProjects({ page: 1, size: 1 }).catch(() => ({ total: 0 })),
          getOnboardingStatus().catch(() => ({ adapters: [] }))
        ]);
        stats.value.skills = (skillRes == null ? void 0 : skillRes.total) || 0;
        stats.value.projects = (projRes == null ? void 0 : projRes.total) || 0;
        const adapters = (obRes == null ? void 0 : obRes.adapters) || [];
        stats.value.toolsTotal = adapters.length;
        stats.value.toolsReady = adapters.filter((a) => a.global_ok).length;
        backendOK.value = true;
      } catch (_) {
        backendOK.value = false;
      }
    }
    onMounted(refreshStats);
    const navItems = computed(() => [
      { key: "skills", label: t("app.nav.skills.label"), icon: "mdi:book-open-variant" },
      { key: "projects", label: t("app.nav.projects.label"), icon: "mdi:folder-multiple-outline" },
      { key: "market", label: t("app.nav.market.label"), icon: "mdi:cart-outline" },
      { key: "audit", label: t("app.nav.audit.label"), icon: "mdi:script-text-outline" },
      { key: "settings", label: t("app.nav.settings.label"), icon: "mdi:cog-outline" }
    ]);
    function switchTab(k) {
      tab.value = k;
      if (k === "audit" || k === "skills") refreshStats();
      if (isMobile.value) sidebarOpen.value = false;
    }
    function onBusEvent(name, payload) {
      if (name === "switch-tab") {
        switchTab(payload);
      }
    }
    function onWindowEvent(e) {
      if ((e == null ? void 0 : e.type) === "skillbox:switch-tab") onBusEvent("switch-tab", e.detail);
    }
    onMounted(() => {
      eventBus.on("switch-tab", onBusEvent);
      window.addEventListener("skillbox:switch-tab", onWindowEvent);
    });
    onUnmounted(() => {
      eventBus.off("switch-tab", onBusEvent);
      window.removeEventListener("skillbox:switch-tab", onWindowEvent);
    });
    return (_ctx, _cache) => {
      var _a;
      return openBlock(), createElementBlock("div", {
        class: normalizeClass(["app-container", isDark.value ? "dark" : ""])
      }, [
        isMobile.value && sidebarOpen.value ? (openBlock(), createElementBlock("div", {
          key: 0,
          class: "fixed inset-0 bg-black/50 z-30 backdrop-blur-sm transition-opacity duration-200",
          onClick: _cache[0] || (_cache[0] = ($event) => sidebarOpen.value = false)
        })) : createCommentVNode("", true),
        createBaseVNode("aside", {
          class: normalizeClass([
            "sidebar flex flex-col z-40",
            "transition-transform duration-300 ease-out",
            isMobile.value ? sidebarOpen.value ? "fixed inset-y-0 left-0 translate-x-0" : "fixed inset-y-0 left-0 -translate-x-full" : "sticky top-0 h-screen"
          ]),
          style: normalizeStyle(!isMobile.value ? { width: sidebarWidth.value + "px" } : {})
        }, [
          createBaseVNode("div", _hoisted_1, [
            createBaseVNode("div", _hoisted_2, [
              createVNode(unref(Icon), {
                icon: "mdi:package-variant-closed",
                width: "24",
                height: "24"
              })
            ]),
            createBaseVNode("div", _hoisted_3, [
              createBaseVNode("span", _hoisted_4, toDisplayString$1(unref(t)("app.brand")), 1)
            ]),
            isMobile.value ? (openBlock(), createElementBlock("button", {
              key: 0,
              class: "mobile-close-btn",
              onClick: _cache[1] || (_cache[1] = ($event) => sidebarOpen.value = false),
              "aria-label": unref(t)("app.closeSidebar")
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:close",
                width: "18",
                height: "18"
              })
            ], 8, _hoisted_5)) : createCommentVNode("", true)
          ]),
          createBaseVNode("nav", _hoisted_6, [
            (openBlock(true), createElementBlock(Fragment, null, renderList(navItems.value, (n) => {
              return openBlock(), createElementBlock("button", {
                key: n.key,
                class: normalizeClass([
                  "nav-item",
                  tab.value === n.key ? "nav-item-active" : ""
                ]),
                onClick: ($event) => switchTab(n.key)
              }, [
                createBaseVNode("span", _hoisted_8, [
                  createVNode(unref(Icon), {
                    icon: n.icon,
                    width: "20",
                    height: "20"
                  }, null, 8, ["icon"])
                ]),
                createBaseVNode("span", _hoisted_9, [
                  createBaseVNode("span", _hoisted_10, toDisplayString$1(n.label), 1)
                ]),
                tab.value === n.key ? (openBlock(), createElementBlock("span", _hoisted_11)) : createCommentVNode("", true)
              ], 10, _hoisted_7);
            }), 128))
          ]),
          createBaseVNode("div", _hoisted_12, [
            createBaseVNode("div", {
              class: normalizeClass(["status-indicator", backendOK.value ? "status-ok" : "status-error"])
            }, [
              createBaseVNode("span", {
                class: normalizeClass(["status-dot", backendOK.value ? "dot-ok" : "dot-error"])
              }, null, 2),
              createBaseVNode("span", _hoisted_13, toDisplayString$1(backendOK.value ? unref(t)("app.backendOk") : unref(t)("app.backendDown")), 1)
            ], 2),
            createBaseVNode("button", {
              class: "theme-toggle",
              onClick: toggleTheme,
              title: isDark.value ? "切换到亮色模式" : "切换到暗黑模式"
            }, [
              createVNode(unref(Icon), {
                icon: isDark.value ? "mdi:weather-sunny" : "mdi:weather-night",
                width: "18",
                height: "18"
              }, null, 8, ["icon"])
            ], 8, _hoisted_14),
            createBaseVNode("button", {
              class: "refresh-btn",
              onClick: refreshStats,
              title: unref(t)("app.refreshStats")
            }, [
              createVNode(unref(Icon), {
                icon: "mdi:refresh",
                width: "16",
                height: "16"
              })
            ], 8, _hoisted_15)
          ]),
          !isMobile.value ? (openBlock(), createElementBlock("div", {
            key: 0,
            class: normalizeClass(["resize-handle", { active: isResizing.value }]),
            onMousedown: startResize,
            title: `侧边栏宽度: ${sidebarWidth.value}px（拖拽调节）`
          }, [..._cache[3] || (_cache[3] = [
            createBaseVNode("div", { class: "resize-grip" }, [
              createBaseVNode("span"),
              createBaseVNode("span")
            ], -1)
          ])], 42, _hoisted_16)) : createCommentVNode("", true)
        ], 6),
        createBaseVNode("main", _hoisted_17, [
          createBaseVNode("header", _hoisted_18, [
            createBaseVNode("div", _hoisted_19, [
              isMobile.value ? (openBlock(), createElementBlock("button", {
                key: 0,
                class: "menu-toggle",
                onClick: _cache[2] || (_cache[2] = ($event) => sidebarOpen.value = true),
                "aria-label": unref(t)("app.openSidebar")
              }, [
                createVNode(unref(Icon), {
                  icon: "mdi:menu",
                  width: "22",
                  height: "22"
                })
              ], 8, _hoisted_20)) : createCommentVNode("", true),
              createBaseVNode("div", _hoisted_21, [
                createBaseVNode("span", _hoisted_22, toDisplayString$1(unref(t)("app.brand")), 1),
                createVNode(unref(Icon), {
                  icon: "mdi:chevron-right",
                  width: "14",
                  height: "14",
                  class: "breadcrumb-sep"
                }),
                createBaseVNode("span", _hoisted_23, toDisplayString$1((_a = navItems.value.find((x) => x.key === tab.value)) == null ? void 0 : _a.label), 1)
              ])
            ]),
            createBaseVNode("div", _hoisted_24, [
              createBaseVNode("div", _hoisted_25, [
                createVNode(unref(Icon), {
                  icon: "mdi:book-open-variant",
                  width: "12",
                  height: "12"
                }),
                createBaseVNode("span", null, toDisplayString$1(unref(t)("app.nav.skills.label")), 1),
                createBaseVNode("strong", null, toDisplayString$1(stats.value.skills), 1)
              ]),
              createBaseVNode("div", _hoisted_26, [
                createVNode(unref(Icon), {
                  icon: "mdi:folder-multiple-outline",
                  width: "12",
                  height: "12"
                }),
                createBaseVNode("span", null, toDisplayString$1(unref(t)("app.nav.projects.label")), 1),
                createBaseVNode("strong", null, toDisplayString$1(stats.value.projects), 1)
              ]),
              createBaseVNode("div", _hoisted_27, [
                createVNode(unref(Icon), {
                  icon: "mdi:tools",
                  width: "12",
                  height: "12"
                }),
                createBaseVNode("span", null, toDisplayString$1(unref(t)("app.toolsLabel")), 1),
                createBaseVNode("strong", null, toDisplayString$1(stats.value.toolsReady) + "/" + toDisplayString$1(stats.value.toolsTotal), 1)
              ])
            ])
          ]),
          createBaseVNode("div", _hoisted_28, [
            tab.value === "projects" ? (openBlock(), createBlock(ProjectsView, { key: 0 })) : tab.value === "skills" ? (openBlock(), createBlock(SkillsView, { key: 1 })) : tab.value === "market" ? (openBlock(), createBlock(MarketView, { key: 2 })) : tab.value === "audit" ? (openBlock(), createBlock(AuditView, { key: 3 })) : tab.value === "settings" ? (openBlock(), createBlock(SettingsView, { key: 4 })) : createCommentVNode("", true)
          ])
        ]),
        createVNode(ToastContainer)
      ], 2);
    };
  }
};
const App = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-ccb3a299"]]);
const zhCN = {
  app: {
    brand: "Skill Box",
    closeSidebar: "关闭侧栏",
    openSidebar: "打开侧栏",
    nav: {
      skills: { label: "技能" },
      projects: { label: "项目" },
      market: { label: "市场" },
      onboarding: { label: "导入技能" },
      audit: { label: "审计" },
      settings: { label: "设置" }
    },
    backendOk: "后端已连接",
    backendDown: "后端断开",
    refreshStats: "刷新统计",
    toolsLabel: "工具"
  },
  common: {
    cancel: "取消",
    create: "创建",
    save: "保存",
    close: "关闭",
    delete: "删除",
    edit: "编辑",
    apply: "应用",
    search: "搜索",
    refresh: "刷新",
    prev: "上一页",
    next: "下一页",
    all: "全部",
    applyFilter: "应用过滤",
    processing: "处理中…",
    none: "—",
    dash: "—",
    confirm: "确认",
    pageOf: "第 {page} / {total} 页 · 共 {count} 条",
    totalCount: "共 {count} 条",
    noData: "该作用域下还没有技能",
    noDataHint: '点右上角"新建"开始,或去首次配置从已装工具导入'
  },
  skills: {
    title: "技能",
    subtitle: "浏览 / 编辑 / 测试 / 落工具 / 打 tag / 回滚。AI 侧栏一键改写 frontmatter 与 body。",
    scopeGlobal: "全局",
    scopeProject: "项目",
    searchPlaceholder: "按名称过滤",
    btnNew: "+ 新建技能",
    btnAiOpen: "打开 AI",
    btnAiClose: "关闭 AI",
    applyBar: {
      target: "应用目标工具:",
      checkUpdates: "检测更新",
      checking: "检测中…",
      updatesAvailable: "{updates} / {total} 可更新",
      allUpToDate: "{total} 个技能已是最新",
      appliedOk: "已把 {name}@{version} 落到 {tool}",
      appliedPartial: "部分失败: {detail}",
      errDefault: "应用失败"
    },
    editor: {
      titleNew: "新建技能",
      titleEdit: "编辑技能",
      name: "名称",
      nameHint: "英文短名,如 review-pr",
      version: "版本",
      versionHint: "0.1.0",
      scope: "作用域",
      projectId: "项目 ID",
      description: "描述",
      descriptionHint: "≥ 10 字符",
      triggers: "触发词",
      triggersHint: "用逗号分隔",
      triggersHintPlaceholder: "review pr, code review",
      body: "正文 (Markdown,frontmatter 会自动拼)",
      // 2026-06-26 新增:作用域区改造 + 适用工具
      scopeGlobal: "全局",
      scopeProject: "项目",
      scopeGlobalHint: "对所有项目可见",
      projectPick: "请选择项目",
      applyTools: "适用工具",
      applyToolsHint: "勾选后会自动启用",
      applyToolsNone: "暂未勾选,创建后只在 skillbox 库中",
      applyToolsSelected: "已选 {n} 个",
      errProjectRequired: "请先选择项目",
      applyAllSuccess: "已在 {n} 个工具上自动启用",
      applyPartialFailed: "{ok}/{total} 个工具启用成功,失败: {fails}",
      // 2026-06-26 新增:编辑保存后同步拷贝到已启用命中的提示
      syncAllSuccess: "已同步到 {n} 个生效位置",
      syncPartialFailed: "{ok}/{total} 个生效位置同步成功",
      syncNone: "「{name}」已保存,但还没在任何工具/项目上启用,需要时去作用域区启用",
      errNameEmpty: "名称不能为空",
      errDescShort: "描述至少 10 个字符",
      errTriggersEmpty: "触发词至少填一个"
    },
    applyHistory: {
      title: "最近应用历史",
      count: "{count} 条",
      undone: "撤销",
      undoing: "撤销中…",
      applied: "已应用",
      rolledBack: "已回滚",
      failed: "失败"
    },
    tag: {
      titlePrefix: "Tag 管理",
      count: "{count} 个 tag",
      createPlaceholder: "tag 名,如 v1.0.0",
      msgPlaceholder: "描述(可选)",
      btnCreate: "打 Tag",
      msgCreated: "已打 tag",
      msgDeleted: "已删除 tag #{id}",
      msgRolledBack: "已回滚(自动打 {pre},恢复 {files} 个文件)",
      selectFirst: "先选一个技能",
      emptyName: "tag 名不能为空",
      confirmDelete: "删除 tag #{id}?file_snapshots 也会一起删。",
      confirmRollback: "回滚到 tag #{id}?会自动打一个 _pre_rollback 隐式 tag,当前状态不会丢失。",
      confirmUndo: "撤销应用 #{id}?将恢复目标目录到应用之前的状态。",
      undoMsg: "已撤销应用 #{id}",
      rollbackTo: "回滚到此",
      rollingBack: "回滚中…",
      diff: "差异",
      seeDiff: "查看差异",
      clear: "清空",
      vsCurrent: "vs 当前",
      current: "当前",
      implicit: " [隐式]",
      resultTitle: "差异结果",
      added: "+{n}",
      removed: "-{n}",
      modified: "~{n}",
      unchanged: "{n} 不变"
    },
    test: {
      title: "最近测试结果",
      errPrefix: "测试失败:",
      passed: "通过",
      failed: "失败",
      errored: "出错",
      skipped: "跳过",
      confirmRun: '对技能 "{name}@{version}" 跑一次测试?(静态 + 脚本 + AI)'
    },
    list: {
      title: "技能",
      colName: "名称",
      colVersion: "版本",
      colSource: "来源",
      colProject: "项目",
      colUpdated: "更新时间",
      colActions: "操作",
      btnApply: "应用",
      applying: "应用中…",
      btnTest: "测试",
      testing: "测试中…",
      btnEdit: "编辑",
      btnTag: "打标签",
      btnDelete: "删除",
      confirmDelete: '确定删除技能 "{name}@{version}" ?',
      emptyTitle: "该作用域下还没有技能",
      emptyHint: '点右上角"+ 新建技能"开始,或去首次配置从已装工具导入',
      // 左右布局新增
      btnNewSkill: "新建",
      btnNewSkillTitle: "新建技能",
      btnImportSkill: "导入",
      btnImportSkillTitle: "从已装工具导入",
      searchTitle: "按名称过滤",
      selectToView: "从左侧选一个技能查看详情",
      noFilesHint: "该技能没有可渲染的正文",
      scopeLabel: "作用域",
      scopeGlobalChip: "全局",
      scopeProjectChip: "项目",
      scopeToolsRow: "工具",
      scopeTargetsRow: "生效位置",
      scopeEmpty: "该技能尚未写入任何工具/位置",
      scopeHitCount: "{n} 处生效",
      scopeSelectToolFirst: "先在「工具」行点选一个工具,再操作生效位置",
      scopeForTool: "对 {tool} 生效",
      scopeToolSelected: "已选 {tool}",
      applyConfirmTitle: "启用作用域",
      applyConfirmMessage: '将 skill "{name}" 复制到 {tool} · {scope}?',
      applySuccess: "已启用:{path}",
      applyFailed: "启用失败:{msg}",
      unapplyConfirmTitle: "停用作用域",
      unapplyConfirmMessage: '将从 {tool} · {scope} 删除 skill "{name}"(走 apply/undo 还原 PreSnapshot),继续?',
      unapplySuccess: "已停用:{path}",
      unapplyFailed: "停用失败:{msg}",
      appliedGlobal: "{tool} 已全局应用",
      applying: "启用中…",
      unapplying: "停用中…",
      tagsEmpty: "还没有标签,点右上角打一个",
      bodyEmpty: "SKILL.md 还没有正文",
      bodyTitle: "正文",
      bodyEditing: "编辑正文 (Markdown)",
      tooltipTest: "测试",
      tooltipTag: "打标签",
      tooltipOpenFolder: "在文件夹打开",
      tooltipDelete: "删除",
      copyPath: "复制路径",
      copied: "已复制",
      openFailed: "打开失败: {msg}",
      goOnboarding: "去导入"
    },
    ai: {
      header: "AI 助手",
      clear: "清空",
      empty: "先选一个预设(优化 frontmatter / 检验描述 / 润色正文 / 查重复 / 安全检查),再发问。",
      hintNoProvider: "暂未配置 AI 提供方或内置预设",
      pickFirst: "请先在上方选一个预设。",
      pickedDedupe: "请在输入框里把要对比的若干技能全文贴进来(每个用 \\n\\n---\\n\\n 分隔),我会给出重叠度评分。",
      pickedPreset: "已选择预设:「{title}」。{description}\n把上下文(可空)和额外要求贴到下方,点发送即可。",
      roleUser: "你",
      roleAssistant: "AI",
      copy: "复制",
      inputPlaceholderHint: "补充说明(可空)",
      inputPlaceholderNoPreset: "先选预设",
      send: "发送",
      stop: "停止",
      noExtraInput: "(无额外输入,只基于上下文)",
      errorTag: "[错误] {msg}"
    }
  },
  projects: {
    title: "项目",
    subtitle: "登记项目根目录,后续技能可绑定到项目作用域,走项目级覆盖。",
    btnNew: "+ 新建项目",
    btnCancel: "取消",
    searchPlaceholder: "按名称过滤",
    formTitle: "新建项目",
    name: "名称",
    nameHint: "显示名,如 My App",
    alias: "别名",
    aliasHint: "唯一别名,英文短码",
    rootPath: "根路径",
    rootPathHint: "项目根绝对路径",
    description: "描述",
    descriptionHint: "可选,描述项目用途",
    errRequired: "名称 / 别名 / 根路径都不能为空",
    listTitle: "项目列表",
    colId: "ID",
    colName: "名称",
    colAlias: "别名",
    colRootPath: "根路径",
    colDescription: "描述",
    colActions: "操作",
    confirmDelete: "确定删除项目 #{id} ?",
    empty: '还没有登记项目。点右上角"+ 新建项目"开始'
  },
  market: {
    title: "三方市场",
    subtitle: "从 skillhub.cn / skills.sh 等三方源拉取技能,直接装到 Skill Box 本地 store。",
    scopeLabel: "作用域:",
    scopeGlobal: "全局 (global)",
    scopeProject: "项目 (暂未启用)",
    searchPlaceholder: "按名称搜索…",
    btnSearch: "搜索",
    btnRefresh: "刷新源",
    refreshing: "刷新中…",
    noSources: "没有可用的源",
    lastRefresh: "上次刷新:拉取 {pulled} · 新增 {inserted} · 更新 {updated}",
    errLoadSources: "源加载失败: {msg}",
    errLoadList: "列表加载失败: {msg}",
    errRefresh: "刷新失败: {msg}",
    errInstall: "安装失败: {msg}",
    okInstalled: "已装:{name} (v{version})",
    installConfirm: '确定把 "{name}" 装到 {scope} 吗?',
    btnInstall: "安装",
    installing: "装中…",
    colName: "名称",
    colVersion: "版本",
    colAuthor: "作者",
    colDescription: "描述",
    colTags: "标签",
    emptyFirstTime: '当前源还没拉过。点 "刷新源" 把三方目录拉到本地。',
    loading: "加载中…"
  },
  onboarding: {
    title: "导入技能",
    subtitle: "扫描本机 5 个 AI 编程工具的技能目录,把发现的技能勾选导入到 Skill Box 自己的 store(全局作用域)。",
    btnRescan: "重新扫描",
    btnRescanning: "扫描中…",
    btnRescanTitle: "重新扫描 5 个 adapter",
    steps: {
      status: "查看状态",
      scan: "扫描 + 勾选",
      done: "完成"
    },
    phase1: {
      title: "工具 adapter 状态",
      total: "共 {n} 个",
      empty: "还没注册 adapter",
      colTool: "工具",
      colId: "ID",
      colGlobalPath: "全局路径",
      colStatus: "状态",
      detected: "已检测到",
      missing: "未找到",
      lastScan: "上次扫描:",
      neverScanned: "从未",
      foundSuffix: "· 共发现 {n} 个技能",
      btnScan: "开始扫描",
      scanning: "扫描中…"
    },
    phase2: {
      title: "扫描结果",
      foundSuffix: "发现 {n} 个技能",
      empty: "这次扫描没找到任何技能",
      emptyHint: '可以点右上角"重新扫描",或先去工具里装一些 skill',
      selectAll: "全选当前",
      selectNone: "清空当前",
      selected: "已选 {sel} / {total}",
      btnBack: "返回上一步",
      btnImport: "导入 {n} 个到 store",
      importing: "导入中…",
      catUser: "用户技能",
      catSystem: "系统技能",
      catSystemHint: "系统级 skill(工具自带 / vendor curated / plugin 内建)只读展示,不能导入",
      catSectionDivider: "以下系统级 skill 不可勾选",
      tagExists: "已存在",
      disabledSystem: "系统级 skill 不能导入",
      disabledExists: "客户端 store 中已存在同名 skill,无法重复导入",
      disabledExclusive: "同名 skill 已被另一个工具勾选,请先取消"
    },
    phase3: {
      title: "导入完成",
      statOk: "成功",
      statErr: "失败",
      statTotal: "总计",
      btnAgain: "再扫一次",
      btnGoSkills: "去技能页查看"
    },
    errScan: "扫描失败: {msg}",
    errImport: "导入失败: {msg}",
    okImport: "导入完成: {ok} 成功 / {failed} 失败"
  },
  audit: {
    title: "审计日志",
    subtitle: "记录所有关键操作的操作者 / 动作 / 目标 / 载荷。第 10 步后端就绪后,这里会自动出现真实数据。",
    statTotal: "总记录数",
    statByAction: "按动作分类",
    statByActor: "按操作者分类",
    placeholderTitle: "第 10 步后端尚未就绪",
    placeholderHint1: "该页面会在 internal/skillpkg/ 导出导入包 + caudit 审计日志控制器完成后自动启用。",
    placeholderHint2: "预计接口:GET /api/skillbox/audit/logs · GET /api/skillbox/audit/stats",
    listTitle: "日志列表",
    filterAction: "动作",
    filterActor: "操作者",
    actorPlaceholder: "用户名",
    filterTargetType: "目标类型",
    targetTypePlaceholder: "技能 / 项目 / ...",
    colId: "ID",
    colTime: "时间",
    colActor: "操作者",
    colAction: "动作",
    colTarget: "目标",
    colPayload: "载荷",
    seeMore: "查看",
    empty: "没有匹配的日志记录"
  },
  settings: {
    title: "设置",
    subtitle: "桌面端偏好(通知 / 全局快捷键 / 启动行为)。Web 端这部分是只读占位。",
    webOnlyHint: "桌面端偏好仅在桌面应用里可见。请用桌面端 / 系统托盘来打开设置。",
    desktop: {
      title: "桌面端偏好",
      subtitle: "需要重启桌面应用生效",
      startMinimized: "启动时最小化到托盘",
      startMinimizedHint: "勾选后,桌面应用启动时不再弹出主窗口,只在托盘留图标",
      notifyEnabled: "启用系统通知",
      notifyEnabledHint: '关闭后,"测试通知"按钮和托盘测试通知都不会发到通知中心',
      shortcutEnabled: "启用全局快捷键",
      shortcutEnabledHint: "关闭后,即使配了组合键也不响应(降级到只走菜单加速键)",
      globalHotkey: "全局快捷键组合",
      globalHotkeyHint: 'V1 仅支持 "Cmd+Shift+S"(macOS);其它组合在后端会拒绝注册',
      globalHotkeyPh: "如 Cmd+Shift+S"
    },
    testNotify: "测试通知",
    testNotifyHint: "向系统通知中心发一条测试横幅,验证授权 / 显示",
    btnTestNotify: "测试通知",
    testTitle: "Skill Box",
    testBody: "这是一条测试通知 — 来自桌面端设置页",
    saved: "已保存",
    errSave: "保存失败: {msg}",
    errNotify: "通知失败: {msg}",
    notifyDisabled: "通知未启用,无法发送",
    notifySent: "通知已发送",
    prefsUnavailable: "偏好服务不可用(可能后端未启动或 prefs 存储未就绪)"
  }
};
const enUS = {
  app: {
    brand: "Skill Box",
    closeSidebar: "Close sidebar",
    openSidebar: "Open sidebar",
    nav: {
      skills: { label: "Skills" },
      projects: { label: "Projects" },
      market: { label: "Market" },
      onboarding: { label: "Import skills" },
      audit: { label: "Audit" },
      settings: { label: "Settings" }
    },
    backendOk: "Backend connected",
    backendDown: "Backend down",
    refreshStats: "Refresh stats",
    toolsLabel: "Tools"
  },
  common: {
    cancel: "Cancel",
    create: "Create",
    save: "Save",
    close: "Close",
    delete: "Delete",
    edit: "Edit",
    apply: "Apply",
    search: "Search",
    refresh: "Refresh",
    prev: "Prev",
    next: "Next",
    all: "All",
    applyFilter: "Apply filter",
    processing: "Processing…",
    none: "—",
    dash: "—",
    confirm: "Confirm",
    pageOf: "Page {page} / {total} · {count} total",
    totalCount: "{count} total",
    noData: "No skills in this scope yet",
    noDataHint: 'Click "+ New" to start, or import from installed tools via Onboarding'
  },
  skills: {
    title: "Skills",
    subtitle: "Browse / edit / test / apply / tag / rollback. The AI side panel rewrites frontmatter and body in one click.",
    scopeGlobal: "Global",
    scopeProject: "Project",
    searchPlaceholder: "Filter by name",
    btnNew: "+ New Skill",
    btnAiOpen: "Open AI",
    btnAiClose: "Close AI",
    applyBar: {
      target: "Apply target tool:",
      checkUpdates: "Check updates",
      checking: "Checking…",
      updatesAvailable: "{updates} / {total} updates available",
      allUpToDate: "{total} skills up to date",
      appliedOk: "Applied {name}@{version} to {tool}",
      appliedPartial: "Partial failure: {detail}",
      errDefault: "Apply failed"
    },
    editor: {
      titleNew: "New Skill",
      titleEdit: "Edit Skill",
      name: "Name",
      nameHint: "short english id, e.g. review-pr",
      version: "Version",
      versionHint: "0.1.0",
      scope: "Scope",
      projectId: "Project ID",
      description: "Description",
      descriptionHint: "min 10 chars",
      triggers: "Triggers",
      triggersHint: "comma-separated",
      triggersHintPlaceholder: "review pr, code review",
      body: "Body (Markdown, frontmatter auto-merged)",
      // 2026-06-26 new: scope refactor + apply tools
      scopeGlobal: "Global",
      scopeProject: "Project",
      scopeGlobalHint: "visible to all projects",
      projectPick: "select a project",
      applyTools: "Apply to tools",
      applyToolsHint: "auto-enable on selected tools",
      applyToolsNone: "none selected, only stored in skillbox",
      applyToolsSelected: "{n} selected",
      errProjectRequired: "please select a project first",
      applyAllSuccess: "auto-enabled on {n} tools",
      applyPartialFailed: "{ok}/{total} tools enabled, failed: {fails}",
      // 2026-06-26 new: edit save sync to enabled locations
      syncAllSuccess: "synced to {n} active locations",
      syncPartialFailed: "{ok}/{total} active locations synced",
      syncNone: '"{name}" saved, but not enabled anywhere yet — enable it from the scope section if needed',
      errNameEmpty: "name is required",
      errDescShort: "description must be at least 10 chars",
      errTriggersEmpty: "at least one trigger is required"
    },
    applyHistory: {
      title: "Recent apply history",
      count: "{count} entries",
      undone: "Undo",
      undoing: "Undoing…",
      applied: "applied",
      rolledBack: "rolled_back",
      failed: "failed"
    },
    tag: {
      titlePrefix: "Tag manager",
      count: "{count} tags",
      createPlaceholder: "tag name, e.g. v1.0.0",
      msgPlaceholder: "message (optional)",
      btnCreate: "Create tag",
      msgCreated: "Tag created",
      msgDeleted: "Deleted tag #{id}",
      msgRolledBack: "Rolled back (auto-tagged {pre}, restored {files} files)",
      selectFirst: "Select a skill first",
      emptyName: "tag name is required",
      confirmDelete: "Delete tag #{id}? file_snapshots will also be deleted.",
      confirmRollback: "Rollback to tag #{id}? an implicit _pre_rollback tag will be created; current state is preserved.",
      confirmUndo: "Undo apply #{id}? The target directory will be restored to its pre-apply state.",
      undoMsg: "Undone apply #{id}",
      rollbackTo: "Rollback here",
      rollingBack: "Rolling back…",
      diff: "Diff",
      seeDiff: "View diff",
      clear: "Clear",
      vsCurrent: "vs current",
      current: "current",
      implicit: " [implicit]",
      resultTitle: "Diff result",
      added: "+{n}",
      removed: "-{n}",
      modified: "~{n}",
      unchanged: "{n} unchanged"
    },
    test: {
      title: "Recent test result",
      errPrefix: "Test failed:",
      passed: "passed",
      failed: "failed",
      errored: "errored",
      skipped: "skipped",
      confirmRun: 'Run test on skill "{name}@{version}"? (static + script + ai)'
    },
    list: {
      title: "Skills",
      colName: "Name",
      colVersion: "Version",
      colSource: "Source",
      colProject: "Project",
      colUpdated: "Updated",
      colActions: "Actions",
      btnApply: "Apply",
      applying: "Applying…",
      btnTest: "Test",
      testing: "Testing…",
      btnEdit: "Edit",
      btnTag: "Tag",
      btnDelete: "Delete",
      confirmDelete: 'Delete skill "{name}@{version}" ?',
      emptyTitle: "No skills in this scope yet",
      emptyHint: 'Click "+ New Skill" to start, or import from installed tools via Onboarding',
      // left/right layout
      btnNewSkill: "New",
      btnNewSkillTitle: "New skill",
      btnImportSkill: "Import",
      btnImportSkillTitle: "Import from installed tools",
      searchTitle: "Filter by name",
      selectToView: "Pick a skill on the left to see details",
      noFilesHint: "This skill has no renderable body",
      scopeLabel: "Scope",
      scopeGlobalChip: "Global",
      scopeProjectChip: "Project",
      scopeToolsRow: "Tools",
      scopeTargetsRow: "Locations",
      scopeEmpty: "This skill is not installed in any tool yet",
      scopeHitCount: "{n} locations",
      scopeSelectToolFirst: 'Pick a tool in the "Tools" row first, then choose a location',
      scopeForTool: "Applies to {tool}",
      scopeToolSelected: "Selected: {tool}",
      applyConfirmTitle: "Enable location",
      applyConfirmMessage: 'Copy skill "{name}" to {tool} · {scope}?',
      applySuccess: "Enabled: {path}",
      applyFailed: "Failed to enable: {msg}",
      unapplyConfirmTitle: "Disable location",
      unapplyConfirmMessage: 'Delete skill "{name}" from {tool} · {scope} (via apply/undo + PreSnapshot restore). Continue?',
      unapplySuccess: "Disabled: {path}",
      unapplyFailed: "Failed to disable: {msg}",
      appliedGlobal: "{tool} applied globally",
      applying: "Enabling…",
      unapplying: "Disabling…",
      tagsEmpty: "No tags yet, click the icon above to create one",
      bodyEmpty: "SKILL.md has no body yet",
      bodyTitle: "Body",
      bodyEditing: "Editing body (Markdown)",
      tooltipTest: "Test",
      tooltipTag: "Tag",
      tooltipOpenFolder: "Open in folder",
      tooltipDelete: "Delete",
      copyPath: "Copy path",
      copied: "Copied",
      openFailed: "Open failed: {msg}",
      goOnboarding: "Go import"
    },
    ai: {
      header: "AI Assistant",
      clear: "Clear",
      empty: "Pick a preset first (optimize frontmatter / check description / polish body / find duplicates / security check), then ask.",
      hintNoProvider: "No AI provider or built-in preset configured",
      pickFirst: "Pick a preset from above first.",
      pickedDedupe: "Paste the skill bodies you want to compare into the input (separate each with \\n\\n---\\n\\n), I will return overlap scores.",
      pickedPreset: "Preset selected: 「{title}」. {description}\nPaste the context (optional) and extra requirements below, then hit Send.",
      roleUser: "You",
      roleAssistant: "AI",
      copy: "Copy",
      inputPlaceholderHint: "Additional notes (optional)",
      inputPlaceholderNoPreset: "Pick a preset first",
      send: "Send",
      stop: "Stop",
      noExtraInput: "(no extra input, context only)",
      errorTag: "[error] {msg}"
    }
  },
  projects: {
    title: "Projects",
    subtitle: "Register project roots; later you can bind skills to a project scope to override global ones.",
    btnNew: "+ New Project",
    btnCancel: "Cancel",
    searchPlaceholder: "Filter by name",
    formTitle: "New project",
    name: "Name",
    nameHint: "display name, e.g. My App",
    alias: "Alias",
    aliasHint: "unique short id",
    rootPath: "Root Path",
    rootPathHint: "absolute path of project root",
    description: "Description",
    descriptionHint: "optional, project purpose",
    errRequired: "name / alias / root_path are all required",
    listTitle: "Projects",
    colId: "ID",
    colName: "Name",
    colAlias: "Alias",
    colRootPath: "Root Path",
    colDescription: "Description",
    colActions: "Actions",
    confirmDelete: "Delete project #{id} ?",
    empty: 'No projects registered yet. Click "+ New Project" to start'
  },
  market: {
    title: "Marketplace",
    subtitle: "Pull skills from 3rd-party sources like skillhub.cn / skills.sh and install them directly into the Skill Box store.",
    scopeLabel: "Scope:",
    scopeGlobal: "Global (global)",
    scopeProject: "Project (not enabled yet)",
    searchPlaceholder: "Search by name…",
    btnSearch: "Search",
    btnRefresh: "Refresh source",
    refreshing: "Refreshing…",
    noSources: "No available sources",
    lastRefresh: "Last refresh: pulled {pulled} · added {inserted} · updated {updated}",
    errLoadSources: "Failed to load sources: {msg}",
    errLoadList: "Failed to load list: {msg}",
    errRefresh: "Refresh failed: {msg}",
    errInstall: "Install failed: {msg}",
    okInstalled: "Installed: {name} (v{version})",
    installConfirm: 'Install "{name}" into {scope} ?',
    btnInstall: "Install",
    installing: "Installing…",
    colName: "name",
    colVersion: "version",
    colAuthor: "author",
    colDescription: "description",
    colTags: "tags",
    emptyFirstTime: 'This source has not been pulled yet. Click "Refresh source" to fetch from the upstream catalog.',
    loading: "Loading…"
  },
  onboarding: {
    title: "Import skills",
    subtitle: "Scan skill directories of the 5 AI coding tools on this machine, pick which ones to import into the Skill Box store (global scope).",
    btnRescan: "Rescan",
    btnRescanning: "Scanning…",
    btnRescanTitle: "Rescan all 5 adapters",
    steps: {
      status: "Status",
      scan: "Scan + select",
      done: "Done"
    },
    phase1: {
      title: "Tool adapter status",
      total: "{n} total",
      empty: "No adapters registered yet",
      colTool: "Tool",
      colId: "ID",
      colGlobalPath: "Global Path",
      colStatus: "Status",
      detected: "Detected",
      missing: "Not found",
      lastScan: "Last scan:",
      neverScanned: "never",
      foundSuffix: "· {n} skills found",
      btnScan: "Start scan",
      scanning: "Scanning…"
    },
    phase2: {
      title: "Scan result",
      foundSuffix: "{n} skills found",
      empty: "No skills found this time.",
      emptyHint: 'Click "Rescan" in the top right, or install some skills first',
      selectAll: "Select current",
      selectNone: "Clear current",
      selected: "{sel} / {total} selected",
      btnBack: "Back",
      btnImport: "Import {n} into store",
      importing: "Importing…",
      catUser: "User skills",
      catSystem: "System skills",
      catSystemHint: "System-level skills (tool-built-in / vendor curated / plugin bundled) are read-only and cannot be imported",
      catSectionDivider: "The following system-level skills cannot be selected",
      tagExists: "Exists",
      disabledSystem: "System-level skills cannot be imported",
      disabledExists: "A skill with the same name already exists in the client store",
      disabledExclusive: "The same skill is already selected from another tool — deselect first"
    },
    phase3: {
      title: "Import complete",
      statOk: "Succeeded",
      statErr: "Failed",
      statTotal: "Total",
      btnAgain: "Scan again",
      btnGoSkills: "View in Skills"
    },
    errScan: "Scan failed: {msg}",
    errImport: "Import failed: {msg}",
    okImport: "Import complete: {ok} succeeded / {failed} failed"
  },
  audit: {
    title: "Audit log",
    subtitle: "Records the actor / action / target / payload of every key operation. After Step 10 is wired up, real data shows up here automatically.",
    statTotal: "Total records",
    statByAction: "By action",
    statByActor: "By actor",
    placeholderTitle: "Step 10 backend not ready yet",
    placeholderHint1: "This page will activate automatically once internal/skillpkg/ export/import and the caudit log controller are wired up.",
    placeholderHint2: "Expected endpoints: GET /api/skillbox/audit/logs · GET /api/skillbox/audit/stats",
    listTitle: "Logs",
    filterAction: "Action",
    filterActor: "Actor",
    actorPlaceholder: "username",
    filterTargetType: "Target Type",
    targetTypePlaceholder: "skill / project / ...",
    colId: "ID",
    colTime: "Time",
    colActor: "Actor",
    colAction: "Action",
    colTarget: "Target",
    colPayload: "Payload",
    seeMore: "View",
    empty: "No matching log records"
  },
  settings: {
    title: "Settings",
    subtitle: "Desktop preferences (notifications / global shortcuts / startup). On Web, this section is a read-only placeholder.",
    webOnlyHint: "Desktop preferences are only visible in the desktop app. Use the system tray to open Settings.",
    desktop: {
      title: "Desktop preferences",
      subtitle: "Requires a desktop app restart to take effect",
      startMinimized: "Start minimized to tray",
      startMinimizedHint: "When enabled, the app starts hidden in the system tray without showing the main window",
      notifyEnabled: "Enable system notifications",
      notifyEnabledHint: 'When off, "Test notification" button and tray test notifications are not delivered to the notification center',
      shortcutEnabled: "Enable global shortcut",
      shortcutEnabledHint: "When off, the registered combo does not respond (falls back to menu accelerator only)",
      globalHotkey: "Global hotkey combo",
      globalHotkeyHint: 'V1 only supports "Cmd+Shift+S" on macOS; other combos are rejected by the backend',
      globalHotkeyPh: "e.g. Cmd+Shift+S"
    },
    testNotify: "Test notification",
    testNotifyHint: "Send a test banner to the system notification center to verify authorization / display",
    btnTestNotify: "Test notification",
    testTitle: "Skill Box",
    testBody: "This is a test notification — sent from the desktop settings page",
    saved: "Saved",
    errSave: "Save failed: {msg}",
    errNotify: "Notification failed: {msg}",
    notifyDisabled: "Notifications are disabled, cannot send",
    notifySent: "Notification sent",
    prefsUnavailable: "Preferences service unavailable (backend may not be running or prefs store not initialized)"
  }
};
const STORAGE_KEY = "skillbox.lang";
function detectLocale() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === "zh-CN" || saved === "en-US") return saved;
  } catch (_) {
  }
  const nav = typeof navigator !== "undefined" && navigator.language || "";
  if (nav.toLowerCase().startsWith("zh")) return "zh-CN";
  return "zh-CN";
}
const i18n = createI18n({
  legacy: false,
  // composition API 模式,必须显式 false
  globalInjection: true,
  locale: detectLocale(),
  fallbackLocale: "zh-CN",
  messages: {
    "zh-CN": zhCN,
    "en-US": enUS
  }
});
async function bootstrap() {
  const pinia = createPinia();
  const app = createApp(App);
  app.use(pinia);
  app.use(i18n);
  const store = useAppStore();
  const runtime = getRuntime();
  store.setRuntime(runtime);
  const base = await resolveBaseURL();
  store.setBaseURL(base);
  const wantDebug = typeof location !== "undefined" && /(^|[?&])(debug|debug=req|debug=1)\b/.test(location.search);
  if (wantDebug) enableDebug();
  window.__APP_CONFIG__ = { baseURL: base, runMode: runtime.runMode, isDesktop: runtime.runMode === "desktop" };
  window.__APP_STORE__ = store;
  dlog("bootstrap ready", {
    runtime,
    baseURL: base,
    debug: isDebug()
  });
  try {
    await http.get("/api/health");
    dlog("health ok");
  } catch (e) {
    console.warn("health check failed,业务接口可能暂时不可用:", e.message);
  }
  app.mount("#app");
}
bootstrap();

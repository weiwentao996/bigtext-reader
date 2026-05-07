export namespace main {
	
	export class FolderFile {
	    path: string;
	    name: string;
	    size: number;
	    modTime: number;
	
	    static createFrom(source: any = {}) {
	        return new FolderFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	    }
	}
	export class OpenResult {
	    meta: reader.FileMeta;
	    page: reader.Page;
	    bookmarks: state.Bookmark[];
	    resumed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OpenResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meta = this.convertValues(source["meta"], reader.FileMeta);
	        this.page = this.convertValues(source["page"], reader.Page);
	        this.bookmarks = this.convertValues(source["bookmarks"], state.Bookmark);
	        this.resumed = source["resumed"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SearchPageResult {
	    page: reader.Page;
	    hitOffset: number;
	    hitByteLength: number;
	    lineStartOffset: number;
	    lineIndex: number;
	    lineCharStart: number;
	    lineCharEnd: number;
	    keyword: string;
	    wrapped: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SearchPageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page = this.convertValues(source["page"], reader.Page);
	        this.hitOffset = source["hitOffset"];
	        this.hitByteLength = source["hitByteLength"];
	        this.lineStartOffset = source["lineStartOffset"];
	        this.lineIndex = source["lineIndex"];
	        this.lineCharStart = source["lineCharStart"];
	        this.lineCharEnd = source["lineCharEnd"];
	        this.keyword = source["keyword"];
	        this.wrapped = source["wrapped"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace reader {
	
	export class FileMeta {
	    path: string;
	    name: string;
	    size: number;
	    // Go type: time
	    modTime: any;
	    encoding: string;
	
	    static createFrom(source: any = {}) {
	        return new FileMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.modTime = this.convertValues(source["modTime"], null);
	        this.encoding = source["encoding"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Page {
	    startOffset: number;
	    endOffset: number;
	    lines: string[];
	    lineStartOffsets: number[];
	    lineEndOffsets: number[];
	    eof: boolean;
	    bof: boolean;
	    hasPrevious: boolean;
	    truncated: boolean;
	    encoding: string;
	    fileSize: number;
	
	    static createFrom(source: any = {}) {
	        return new Page(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startOffset = source["startOffset"];
	        this.endOffset = source["endOffset"];
	        this.lines = source["lines"];
	        this.lineStartOffsets = source["lineStartOffsets"];
	        this.lineEndOffsets = source["lineEndOffsets"];
	        this.eof = source["eof"];
	        this.bof = source["bof"];
	        this.hasPrevious = source["hasPrevious"];
	        this.truncated = source["truncated"];
	        this.encoding = source["encoding"];
	        this.fileSize = source["fileSize"];
	    }
	}
	export class PageWindow {
	    pages: Page[];
	    anchor: Page;
	    fileSize: number;
	    encoding: string;
	
	    static createFrom(source: any = {}) {
	        return new PageWindow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pages = this.convertValues(source["pages"], Page);
	        this.anchor = this.convertValues(source["anchor"], Page);
	        this.fileSize = source["fileSize"];
	        this.encoding = source["encoding"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SearchHit {
	    index: number;
	    offset: number;
	    byteLength: number;
	    lineStart: number;
	    lineEnd: number;
	    lineNumber: number;
	    linePreview: string;
	    lineCharStart: number;
	    lineCharEnd: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.offset = source["offset"];
	        this.byteLength = source["byteLength"];
	        this.lineStart = source["lineStart"];
	        this.lineEnd = source["lineEnd"];
	        this.lineNumber = source["lineNumber"];
	        this.linePreview = source["linePreview"];
	        this.lineCharStart = source["lineCharStart"];
	        this.lineCharEnd = source["lineCharEnd"];
	    }
	}
	export class SearchHitPreviewPage {
	    searchId: string;
	    keyword: string;
	    offset: number;
	    limit: number;
	    total: number;
	    hits: SearchHit[];
	
	    static createFrom(source: any = {}) {
	        return new SearchHitPreviewPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.searchId = source["searchId"];
	        this.keyword = source["keyword"];
	        this.offset = source["offset"];
	        this.limit = source["limit"];
	        this.total = source["total"];
	        this.hits = this.convertValues(source["hits"], SearchHit);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SearchSessionSummary {
	    searchId: string;
	    keyword: string;
	    total: number;
	    fileSize: number;
	    encoding: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchSessionSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.searchId = source["searchId"];
	        this.keyword = source["keyword"];
	        this.total = source["total"];
	        this.fileSize = source["fileSize"];
	        this.encoding = source["encoding"];
	    }
	}
	export class SearchSummary {
	    keyword: string;
	    total: number;
	    hits: SearchHit[];
	    truncated: boolean;
	    limit: number;
	    fileSize: number;
	    encoding: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.keyword = source["keyword"];
	        this.total = source["total"];
	        this.hits = this.convertValues(source["hits"], SearchHit);
	        this.truncated = source["truncated"];
	        this.limit = source["limit"];
	        this.fileSize = source["fileSize"];
	        this.encoding = source["encoding"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace state {
	
	export class Bookmark {
	    name: string;
	    offset: number;
	    note?: string;
	    createdAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Bookmark(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.offset = source["offset"];
	        this.note = source["note"];
	        this.createdAt = source["createdAt"];
	    }
	}

}


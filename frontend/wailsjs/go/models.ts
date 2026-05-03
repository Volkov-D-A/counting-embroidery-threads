export namespace threadcalc {
	
	export class CodeCorrection {
	    from: string;
	    to: string;
	
	    static createFrom(source: any = {}) {
	        return new CodeCorrection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	    }
	}
	export class ThreadResult {
	    code: string;
	    colorName: string;
	    colorHex: string;
	    paletteFound: boolean;
	    meters: number;
	    skeins: number;
	    notes: string[];
	
	    static createFrom(source: any = {}) {
	        return new ThreadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.colorName = source["colorName"];
	        this.colorHex = source["colorHex"];
	        this.paletteFound = source["paletteFound"];
	        this.meters = source["meters"];
	        this.skeins = source["skeins"];
	        this.notes = source["notes"];
	    }
	}
	export class ImportResult {
	    cancelled: boolean;
	    filePath: string;
	    fileName: string;
	    encoding: string;
	    rowsImported: number;
	    beadRowsIgnored: number;
	    totalMeters: number;
	    totalSkeins: number;
	    skeinLengthMeters: number;
	    items: ThreadResult[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cancelled = source["cancelled"];
	        this.filePath = source["filePath"];
	        this.fileName = source["fileName"];
	        this.encoding = source["encoding"];
	        this.rowsImported = source["rowsImported"];
	        this.beadRowsIgnored = source["beadRowsIgnored"];
	        this.totalMeters = source["totalMeters"];
	        this.totalSkeins = source["totalSkeins"];
	        this.skeinLengthMeters = source["skeinLengthMeters"];
	        this.items = this.convertValues(source["items"], ThreadResult);
	        this.warnings = source["warnings"];
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


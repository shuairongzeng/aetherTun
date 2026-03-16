export namespace config {
	
	export class BasicProxySettings {
	    Host: string;
	    Port: number;
	    Type: string;
	
	    static createFrom(source: any = {}) {
	        return new BasicProxySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Host = source["Host"];
	        this.Port = source["Port"];
	        this.Type = source["Type"];
	    }
	}
	export class OnboardingState {
	    configExists: boolean;
	    isDefaultProxyConfig: boolean;
	    shouldShowOnboarding: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OnboardingState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configExists = source["configExists"];
	        this.isDefaultProxyConfig = source["isDefaultProxyConfig"];
	        this.shouldShowOnboarding = source["shouldShowOnboarding"];
	    }
	}

}

export namespace logs {
	
	export class Entry {
	    // Go type: time
	    time: any;
	    level: string;
	    source: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = this.convertValues(source["time"], null);
	        this.level = source["level"];
	        this.source = source["source"];
	        this.message = source["message"];
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

export namespace main {
	
	export class AppStatus {
	    phase: string;
	    proxyEndpoint?: string;
	    tunAdapterName?: string;
	    lastErrorCode?: string;
	    lastErrorText?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.proxyEndpoint = source["proxyEndpoint"];
	        this.tunAdapterName = source["tunAdapterName"];
	        this.lastErrorCode = source["lastErrorCode"];
	        this.lastErrorText = source["lastErrorText"];
	    }
	}
	export class SaveBasicProxySettingsResult {
	    settings: config.BasicProxySettings;
	    requiresRestart: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SaveBasicProxySettingsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], config.BasicProxySettings);
	        this.requiresRestart = source["requiresRestart"];
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

export namespace runtime {
	
	export class RuntimeStatus {
	    LastErrorCode: string;
	    Phase: string;
	    LastErrorText: string;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.LastErrorCode = source["LastErrorCode"];
	        this.Phase = source["Phase"];
	        this.LastErrorText = source["LastErrorText"];
	    }
	}

}


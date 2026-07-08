export namespace config {
	
	export class AppConfig {
	    theme: string;
	    accent: string;
	    minimize_to_tray: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.accent = source["accent"];
	        this.minimize_to_tray = source["minimize_to_tray"];
	    }
	}
	export class WindowConfig {
	    x: number;
	    y: number;
	    saved: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WindowConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.saved = source["saved"];
	    }
	}
	export class PomodoroConfig {
	    work_duration: number;
	    short_break: number;
	    long_break: number;
	    sessions_before_long: number;
	
	    static createFrom(source: any = {}) {
	        return new PomodoroConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.work_duration = source["work_duration"];
	        this.short_break = source["short_break"];
	        this.long_break = source["long_break"];
	        this.sessions_before_long = source["sessions_before_long"];
	    }
	}
	export class DatabaseConfig {
	    type: string;
	    dsn: string;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.dsn = source["dsn"];
	    }
	}
	export class Config {
	    database: DatabaseConfig;
	    pomodoro: PomodoroConfig;
	    app: AppConfig;
	    window: WindowConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.database = this.convertValues(source["database"], DatabaseConfig);
	        this.pomodoro = this.convertValues(source["pomodoro"], PomodoroConfig);
	        this.app = this.convertValues(source["app"], AppConfig);
	        this.window = this.convertValues(source["window"], WindowConfig);
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
	
	export class UpdateInfo {
	    available: boolean;
	    current_version: string;
	    latest_version: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.current_version = source["current_version"];
	        this.latest_version = source["latest_version"];
	        this.url = source["url"];
	    }
	}

}

export namespace models {
	
	export class PomodoroSession {
	    ID: number;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	    // Go type: gorm
	    DeletedAt: any;
	    task_id: number;
	    type: string;
	    duration: number;
	    completed: boolean;
	    // Go type: time
	    started_at: any;
	    // Go type: time
	    completed_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new PomodoroSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	        this.DeletedAt = this.convertValues(source["DeletedAt"], null);
	        this.task_id = source["task_id"];
	        this.type = source["type"];
	        this.duration = source["duration"];
	        this.completed = source["completed"];
	        this.started_at = this.convertValues(source["started_at"], null);
	        this.completed_at = this.convertValues(source["completed_at"], null);
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
	export class Task {
	    ID: number;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	    // Go type: gorm
	    DeletedAt: any;
	    title: string;
	    description: string;
	    priority: string;
	    status: string;
	    tags: string;
	    // Go type: time
	    due_date?: any;
	    pomodoro_count: number;
	    position: number;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	        this.DeletedAt = this.convertValues(source["DeletedAt"], null);
	        this.title = source["title"];
	        this.description = source["description"];
	        this.priority = source["priority"];
	        this.status = source["status"];
	        this.tags = source["tags"];
	        this.due_date = this.convertValues(source["due_date"], null);
	        this.pomodoro_count = source["pomodoro_count"];
	        this.position = source["position"];
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

export namespace services {
	
	export class PomodoroStats {
	    sessions_completed: number;
	    work_sessions: number;
	    total_focus_minutes: number;
	
	    static createFrom(source: any = {}) {
	        return new PomodoroStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessions_completed = source["sessions_completed"];
	        this.work_sessions = source["work_sessions"];
	        this.total_focus_minutes = source["total_focus_minutes"];
	    }
	}
	export class TimerState {
	    seconds_remaining: number;
	    session_type: string;
	    is_running: boolean;
	    active_task_id: number;
	
	    static createFrom(source: any = {}) {
	        return new TimerState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seconds_remaining = source["seconds_remaining"];
	        this.session_type = source["session_type"];
	        this.is_running = source["is_running"];
	        this.active_task_id = source["active_task_id"];
	    }
	}

}


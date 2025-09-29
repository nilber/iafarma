import { API_BASE_URL } from '@/lib/api/client';

// Types for fingerprint data
export interface FingerprintData {
  screenResolution: string;
  colorDepth: number;
  timezone: string;
  language: string;
  platform: string;
  cookiesEnabled: boolean | null;
  localStorageEnabled: boolean | null;
  sessionStorageEnabled: boolean | null;
  canvasFingerprint: string;
  audioFingerprint: string;
  webglFingerprint: string;
  plugins: string[];
  fonts: string[];
  navigator: Record<string, any>;
  mouseMovements: Record<string, any>;
  keyboardPatterns: Record<string, any>;
  scrollBehavior: Record<string, any>;
}

export interface BehaviorTracking {
  mouseMovements: Array<{ x: number; y: number; timestamp: number }>;
  keyboardPatterns: {
    typingSpeed: number;
    averageInterval: number;
    patterns: number[];
  };
  scrollBehavior: {
    averageSpeed: number;
    patterns: Array<{ delta: number; timestamp: number }>;
  };
}

export class FingerprintCollector {
  private static instance: FingerprintCollector;
  private sessionFingerprintSent: boolean = false;
  private sessionId: string = '';
  private behaviorData: BehaviorTracking = {
    mouseMovements: [],
    keyboardPatterns: {
      typingSpeed: 0,
      averageInterval: 0,
      patterns: []
    },
    scrollBehavior: {
      averageSpeed: 0,
      patterns: []
    }
  };
  private isTracking = false;
  private lastKeyPress = 0;
  private keyIntervals: number[] = [];

  private constructor() {
    // Initialize session
    this.initializeSession();
  }
  
  private initializeSession(): void {
    // Check if we have a session ID already
    this.sessionId = sessionStorage.getItem('fp_session_id') || '';
    
    if (!this.sessionId) {
      // Generate new session ID
      this.sessionId = 'session_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
      sessionStorage.setItem('fp_session_id', this.sessionId);
      this.sessionFingerprintSent = false;
    } else {
      // Check if we already sent fingerprint in this session
      this.sessionFingerprintSent = sessionStorage.getItem('fp_sent') === 'true';
    }
  }

  public static getInstance(): FingerprintCollector {
    if (!FingerprintCollector.instance) {
      FingerprintCollector.instance = new FingerprintCollector();
    }
    return FingerprintCollector.instance;
  }

  // Start collecting behavioral data
  public startTracking(): void {
    if (this.isTracking) return;
    this.isTracking = true;

    // Track mouse movements
    document.addEventListener('mousemove', this.trackMouseMovement.bind(this));
    
    // Track keyboard patterns
    document.addEventListener('keydown', this.trackKeyboardPattern.bind(this));
    
    // Track scroll behavior
    document.addEventListener('wheel', this.trackScrollBehavior.bind(this));
  }

  // Stop collecting behavioral data
  public stopTracking(): void {
    if (!this.isTracking) return;
    this.isTracking = false;

    document.removeEventListener('mousemove', this.trackMouseMovement.bind(this));
    document.removeEventListener('keydown', this.trackKeyboardPattern.bind(this));
    document.removeEventListener('wheel', this.trackScrollBehavior.bind(this));
  }

  private trackMouseMovement(event: MouseEvent): void {
    if (this.behaviorData.mouseMovements.length > 100) {
      this.behaviorData.mouseMovements.shift(); // Keep only last 100 movements
    }
    
    this.behaviorData.mouseMovements.push({
      x: event.clientX,
      y: event.clientY,
      timestamp: Date.now()
    });
  }

  private trackKeyboardPattern(event: KeyboardEvent): void {
    const now = Date.now();
    if (this.lastKeyPress > 0) {
      const interval = now - this.lastKeyPress;
      this.keyIntervals.push(interval);
      
      if (this.keyIntervals.length > 50) {
        this.keyIntervals.shift(); // Keep only last 50 intervals
      }
      
      // Calculate average
      this.behaviorData.keyboardPatterns.averageInterval = 
        this.keyIntervals.reduce((a, b) => a + b, 0) / this.keyIntervals.length;
    }
    this.lastKeyPress = now;
  }

  private trackScrollBehavior(event: WheelEvent): void {
    if (this.behaviorData.scrollBehavior.patterns.length > 50) {
      this.behaviorData.scrollBehavior.patterns.shift(); // Keep only last 50 scrolls
    }
    
    this.behaviorData.scrollBehavior.patterns.push({
      delta: event.deltaY,
      timestamp: Date.now()
    });

    // Calculate average speed
    const recentPatterns = this.behaviorData.scrollBehavior.patterns.slice(-10);
    this.behaviorData.scrollBehavior.averageSpeed = 
      recentPatterns.reduce((sum, p) => sum + Math.abs(p.delta), 0) / recentPatterns.length;
  }

  // Collect complete fingerprint
  public async collectFingerprint(): Promise<FingerprintData> {
    const fingerprint: FingerprintData = {
      screenResolution: `${screen.width}x${screen.height}`,
      colorDepth: screen.colorDepth,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      language: navigator.language,
      platform: navigator.platform,
      cookiesEnabled: this.checkCookiesEnabled(),
      localStorageEnabled: this.checkLocalStorageEnabled(),
      sessionStorageEnabled: this.checkSessionStorageEnabled(),
      canvasFingerprint: this.generateCanvasFingerprint(),
      audioFingerprint: await this.generateAudioFingerprint(),
      webglFingerprint: this.generateWebGLFingerprint(),
      plugins: this.getPlugins(),
      fonts: await this.detectFonts(),
      navigator: this.getNavigatorInfo(),
      mouseMovements: this.behaviorData.mouseMovements.reduce((acc, curr, index) => {
        if (index % 10 === 0) acc[`movement_${index}`] = curr; // Sample every 10th movement
        return acc;
      }, {} as Record<string, any>),
      keyboardPatterns: this.behaviorData.keyboardPatterns,
      scrollBehavior: this.behaviorData.scrollBehavior
    };

    return fingerprint;
  }

  private checkCookiesEnabled(): boolean | null {
    try {
      document.cookie = 'test_cookie=1';
      const cookieEnabled = document.cookie.indexOf('test_cookie') !== -1;
      document.cookie = 'test_cookie=1; expires=Thu, 01-Jan-1970 00:00:01 GMT'; // Delete test cookie
      return cookieEnabled;
    } catch (e) {
      return null;
    }
  }

  private checkLocalStorageEnabled(): boolean | null {
    try {
      const testKey = '__localStorage_test__';
      localStorage.setItem(testKey, 'test');
      localStorage.removeItem(testKey);
      return true;
    } catch (e) {
      return false;
    }
  }

  private checkSessionStorageEnabled(): boolean | null {
    try {
      const testKey = '__sessionStorage_test__';
      sessionStorage.setItem(testKey, 'test');
      sessionStorage.removeItem(testKey);
      return true;
    } catch (e) {
      return false;
    }
  }

  private generateCanvasFingerprint(): string {
    try {
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      if (!ctx) return '';

      // Draw a complex pattern
      ctx.textBaseline = 'top';
      ctx.font = '14px Arial';
      ctx.fillStyle = '#f60';
      ctx.fillRect(125, 1, 62, 20);
      ctx.fillStyle = '#069';
      ctx.fillText('Canvas fingerprint', 2, 15);
      ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
      ctx.fillText('Canvas fingerprint', 4, 17);

      // Add some geometric shapes
      ctx.beginPath();
      ctx.arc(50, 50, 20, 0, 2 * Math.PI);
      ctx.fillStyle = '#ff0000';
      ctx.fill();

      return canvas.toDataURL();
    } catch (e) {
      return '';
    }
  }

  private async generateAudioFingerprint(): Promise<string> {
    try {
      const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();
      const oscillator = audioContext.createOscillator();
      const analyser = audioContext.createAnalyser();
      const gainNode = audioContext.createGain();

      oscillator.type = 'triangle';
      oscillator.frequency.value = 1000;
      
      gainNode.gain.value = 0.01; // Very low volume
      oscillator.connect(gainNode);
      gainNode.connect(analyser);
      
      oscillator.start();
      
      const bufferLength = analyser.frequencyBinCount;
      const dataArray = new Uint8Array(bufferLength);
      analyser.getByteFrequencyData(dataArray);
      
      oscillator.stop();
      audioContext.close();

      return Array.from(dataArray).slice(0, 20).join(',');
    } catch (e) {
      return '';
    }
  }

  private generateWebGLFingerprint(): string {
    try {
      const canvas = document.createElement('canvas');
      const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl') as WebGLRenderingContext;
      if (!gl) return '';

      const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
      if (!debugInfo) return '';

      const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
      const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
      
      return `${vendor}|${renderer}`;
    } catch (e) {
      return '';
    }
  }

  private getPlugins(): string[] {
    const plugins: string[] = [];
    for (let i = 0; i < navigator.plugins.length; i++) {
      plugins.push(navigator.plugins[i].name);
    }
    return plugins;
  }

  private async detectFonts(): Promise<string[]> {
    const fonts = [
      'Arial', 'Helvetica', 'Times New Roman', 'Courier New', 'Verdana',
      'Georgia', 'Comic Sans MS', 'Trebuchet MS', 'Arial Black', 'Impact'
    ];
    
    const detectedFonts: string[] = [];
    
    for (const font of fonts) {
      if (await this.isFontAvailable(font)) {
        detectedFonts.push(font);
      }
    }
    
    return detectedFonts;
  }

  private async isFontAvailable(fontName: string): Promise<boolean> {
    try {
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      if (!ctx) return false;

      const text = 'mmmmmmmmmmlli';
      
      ctx.font = `72px monospace`;
      const baselineWidth = ctx.measureText(text).width;
      
      ctx.font = `72px ${fontName}, monospace`;
      const testWidth = ctx.measureText(text).width;
      
      return baselineWidth !== testWidth;
    } catch (e) {
      return false;
    }
  }

  private getNavigatorInfo(): Record<string, any> {
    return {
      userAgent: navigator.userAgent,
      language: navigator.language,
      languages: navigator.languages,
      platform: navigator.platform,
      cookieEnabled: navigator.cookieEnabled,
      doNotTrack: navigator.doNotTrack,
      hardwareConcurrency: navigator.hardwareConcurrency,
      maxTouchPoints: navigator.maxTouchPoints,
      onLine: navigator.onLine,
      vendor: navigator.vendor,
      vendorSub: navigator.vendorSub,
      productSub: navigator.productSub,
      appName: navigator.appName,
      appVersion: navigator.appVersion,
      oscpu: (navigator as any).oscpu
    };
  }

  // Send fingerprint to server
  public async sendFingerprint(endpoint: string = `${API_BASE_URL}/admin/security/collect-fingerprint`): Promise<void> {
    try {
      const fingerprint = await this.collectFingerprint();
      
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('access_token')}`
        },
        body: JSON.stringify(fingerprint)
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      
      if (result.success && result.is_suspicious) {
        console.warn('Security alert: Suspicious activity detected');
      }
      
    } catch (error) {
      console.error('Failed to send fingerprint:', error);
      // Don't throw error to prevent breaking the application
    }
  }

  // Auto-collect on page load and specific events
  public enableAutoCollection(): void {
    // Collect on page load
    if (document.readyState === 'complete') {
      this.autoCollect();
    } else {
      window.addEventListener('load', () => this.autoCollect());
    }

    // Collect on focus change (potential session switching)
    window.addEventListener('focus', () => this.autoCollect());
    
    // Collect on visibility change
    document.addEventListener('visibilitychange', () => {
      if (!document.hidden) {
        this.autoCollect();
      }
    });
  }

  private async autoCollect(): Promise<void> {
    try {
      // Check if already sent fingerprint in this session
      if (this.sessionFingerprintSent) {
        console.log('Fingerprint já enviado nesta sessão:', this.sessionId);
        return;
      }

      // Start behavior tracking
      this.startTracking();
      
      // Wait a bit to collect some behavioral data
      setTimeout(async () => {
        await this.sendFingerprint();
        
        // Mark as sent for this session
        this.sessionFingerprintSent = true;
        sessionStorage.setItem('fp_sent', 'true');
        console.log('Fingerprint enviado para sessão:', this.sessionId);
        
        // Stop tracking after sending
        this.stopTracking();
      }, 3000); // Wait 3 seconds to collect behavioral data
      
    } catch (error) {
      console.error('Error in auto-collect:', error);
    }
  }
  
  // Method to manually reset session (useful for testing or logout)
  public resetSession(): void {
    sessionStorage.removeItem('fp_session_id');
    sessionStorage.removeItem('fp_sent');
    this.initializeSession();
    console.log('Sessão de fingerprint resetada');
  }
}

// Export singleton instance
export const fingerprintCollector = FingerprintCollector.getInstance();
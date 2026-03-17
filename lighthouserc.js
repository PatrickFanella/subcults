module.exports = {
  ci: {
    collect: {
      // Build directory containing static files to test
      staticDistDir: './web/dist',
      
      // Number of runs per URL (more runs = more stable results)
      numberOfRuns: 3,
      
      // URLs to test (relative to staticDistDir)
      url: [
        'http://localhost:9000/index.html',
      ],
      
      // Settings for the Chrome instance
      settings: {
        // Use desktop emulation for consistent results
        preset: 'desktop',
        
        // Disable throttling for faster builds (can enable for more realistic results)
        throttling: {
          rttMs: 40,
          throughputKbps: 10240,
          cpuSlowdownMultiplier: 1,
        },
      },
    },
    
    assert: {
      // Performance budgets based on Core Web Vitals
      assertions: {
        // First Contentful Paint: <1.0s
        'first-contentful-paint': ['error', { maxNumericValue: 1000 }],
        
        // Largest Contentful Paint: <2.5s
        'largest-contentful-paint': ['error', { maxNumericValue: 2500 }],
        
        // Cumulative Layout Shift: <0.1
        'cumulative-layout-shift': ['error', { maxNumericValue: 0.1 }],
        
        // Time to First Byte: <600ms
        'server-response-time': ['error', { maxNumericValue: 600 }],
        
        // Speed Index: <3.0s (general performance indicator)
        'speed-index': ['error', { maxNumericValue: 3000 }],
        
        // Total Blocking Time: <200ms (related to INP)
        'total-blocking-time': ['error', { maxNumericValue: 200 }],
        
        // Interactive: <3.8s
        'interactive': ['error', { maxNumericValue: 3800 }],
        
        // Performance score: >90
        'categories:performance': ['error', { minScore: 0.9 }],
        
        // Accessibility score: >90
        'categories:accessibility': ['warn', { minScore: 0.9 }],
        
        // Best practices score: >90
        'categories:best-practices': ['warn', { minScore: 0.9 }],
        
        // Image optimization audits
        'modern-image-formats': ['warn', { minScore: 0.9 }], // WebP usage
        'uses-responsive-images': ['warn', { minScore: 0.9 }], // Properly sized images
        'offscreen-images': ['warn', { minScore: 0.9 }], // Lazy loading
        'uses-optimized-images': ['warn', { minScore: 0.9 }], // Image compression
        'unsized-images': ['error', { minScore: 1.0 }], // Images must have width/height
        
        // Bundle size budgets
        'resource-summary:script:size': ['warn', { maxNumericValue: 300000 }], // 300KB
        'resource-summary:stylesheet:size': ['warn', { maxNumericValue: 50000 }], // 50KB
        'resource-summary:document:size': ['warn', { maxNumericValue: 20000 }], // 20KB
        'resource-summary:image:size': ['warn', { maxNumericValue: 500000 }], // 500KB total images
        'resource-summary:total:size': ['warn', { maxNumericValue: 1000000 }], // 1MB total payload (scripts + styles + images). Increased from 500KB to support image-heavy pages with optimized images; separate image budget (500KB) controls image-specific size.
      },
      
      // Fail build on any errors (not warnings)
      level: 'error',
      
      // Allow 10% regression on numeric values
      budgetThresholds: {
        regression: 0.1, // 10% regression threshold
      },
    },
    
    upload: {
      // Store results in temporary storage (can configure external storage later)
      target: 'temporary-public-storage',
      
      // GitHub App integration (for PR comments)
      // This requires LHCI_GITHUB_APP_TOKEN environment variable
      // Uncomment to enable GitHub integration:
      // target: 'lhci',
      // serverBaseUrl: 'https://your-lhci-server.example.com',
      // token: process.env.LHCI_BUILD_TOKEN,
    },
    
    server: {
      // Optional: Configure for self-hosted LHCI server
      // Uncomment and configure when ready to set up persistent storage
      // baseUrl: 'https://your-lhci-server.example.com',
    },
  },
};

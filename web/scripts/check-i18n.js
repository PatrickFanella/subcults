#!/usr/bin/env node

/**
 * Translation Key Extraction Script
 * Finds all translation keys used in the codebase and validates against locale files
 */

import { readFileSync, readdirSync, statSync, existsSync } from 'fs';
import { join, relative } from 'path';
import { fileURLToPath } from 'url';
import { dirname } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const ROOT_DIR = join(__dirname, '..');
const SRC_DIR = join(ROOT_DIR, 'src');
const LOCALES_DIR = join(ROOT_DIR, 'public', 'locales');

// Load NAMESPACES from i18n.ts to ensure consistency
function loadNamespaces() {
  const i18nPath = join(SRC_DIR, 'i18n.ts');
  const content = readFileSync(i18nPath, 'utf-8');
  const match = content.match(/export const NAMESPACES = \[([^\]]+)\]/);
  if (match) {
    // Parse the array values
    return match[1]
      .split(',')
      .map(s => s.trim().replace(/['"]/g, ''));
  }
  throw new Error('Could not load NAMESPACES from i18n.ts');
}

const NAMESPACES = loadNamespaces();

/**
 * Remove comments from code
 */
function removeComments(content) {
  // Remove single-line comments
  content = content.replace(/\/\/.*$/gm, '');
  // Remove multi-line comments
  content = content.replace(/\/\*[\s\S]*?\*\//g, '');
  return content;
}

/**
 * Recursively find all TypeScript/TSX files
 */
function findSourceFiles(dir) {
  const files = [];
  
  function traverse(currentDir) {
    const items = readdirSync(currentDir);
    
    for (const item of items) {
      const fullPath = join(currentDir, item);
      const stat = statSync(fullPath);
      
      if (stat.isDirectory()) {
        if (!['node_modules', 'dist', 'build', '.git'].includes(item)) {
          traverse(fullPath);
        }
      } else if (stat.isFile() && /\.(ts|tsx)$/.test(item) && !item.endsWith('.test.ts') && !item.endsWith('.test.tsx')) {
        files.push(fullPath);
      }
    }
  }
  
  traverse(dir);
  return files;
}

/**
 * Extract translation keys from a file with namespace awareness
 * Handles multiple useTranslation calls with different namespaces
 */
function extractKeysFromFile(filePath) {
  let content = readFileSync(filePath, 'utf-8');
  
  // Remove comments before processing
  content = removeComments(content);
  
  const keys = [];
  
  // Find all useTranslation calls and their namespaces
  const useTranslationPattern = /useTranslation\s*\(\s*['"`]([^'"`]+)['"`]\s*\)/g;
  const namespaceUsages = [];
  let match;
  
  while ((match = useTranslationPattern.exec(content)) !== null) {
    namespaceUsages.push({
      namespace: match[1],
      position: match.index
    });
  }
  
  // If no explicit namespace found, use 'common' as default
  const defaultNamespace = namespaceUsages.length > 0 ? namespaceUsages[0].namespace : 'common';
  
  // For files with multiple useTranslation calls, we use the first one as default
  // This is a heuristic; ideally we'd do proper AST parsing to track which t() belongs to which hook
  // But for most cases, components use a single namespace
  
  // Pattern for t() calls
  const tPattern = /\bt\s*\(\s*['"`]([^'"`]+)['"`]/g;
  
  while ((match = tPattern.exec(content)) !== null) {
    const key = match[1];
    if (key.includes(':')) {
      keys.push(key);
    } else {
      keys.push(`${defaultNamespace}:${key}`);
    }
  }
  
  // Pattern for template strings
  const tTemplatePattern = /\bt\s*\(\s*`([^`]*\$\{[^}]+\}[^`]*)`/g;
  
  while ((match = tTemplatePattern.exec(content)) !== null) {
    const key = match[1];
    const cleanKey = key.replace(/\$\{[^}]+\}/g, '${var}');
    if (cleanKey.includes(':')) {
      keys.push(cleanKey);
    } else {
      keys.push(`${defaultNamespace}:${cleanKey}`);
    }
  }
  
  // Pattern for i18nKey prop
  const i18nKeyPattern = /i18nKey\s*=\s*['"`]([^'"`]+)['"`]/g;
  
  while ((match = i18nKeyPattern.exec(content)) !== null) {
    const key = match[1];
    if (key.includes(':')) {
      keys.push(key);
    } else {
      keys.push(`${defaultNamespace}:${key}`);
    }
  }
  
  return keys;
}

/**
 * Load translations from a locale file
 */
function loadTranslations(locale, namespace) {
  const filePath = join(LOCALES_DIR, locale, `${namespace}.json`);
  
  if (!existsSync(filePath)) {
    return {};
  }
  
  try {
    const content = readFileSync(filePath, 'utf-8');
    return JSON.parse(content);
  } catch (error) {
    console.error(`Error parsing ${filePath}:`, error.message);
    return {};
  }
}

/**
 * Check if a key exists in translations object
 */
function hasKey(translations, keyPath) {
  const parts = keyPath.split('.');
  let current = translations;
  
  for (const part of parts) {
    if (part.includes('${')) {
      continue;
    }
    if (typeof current !== 'object' || current === null || !(part in current)) {
      return false;
    }
    current = current[part];
  }
  
  return true;
}

/**
 * Parse namespace:key format
 */
function parseKey(key) {
  const colonIndex = key.indexOf(':');
  
  if (colonIndex === -1) {
    return { namespace: 'common', key };
  }
  
  return {
    namespace: key.substring(0, colonIndex),
    key: key.substring(colonIndex + 1),
  };
}

/**
 * Main function
 */
function main() {
  console.log('🔍 Scanning for translation keys...\n');
  
  const sourceFiles = findSourceFiles(SRC_DIR);
  console.log(`Found ${sourceFiles.length} source files\n`);
  
  const allKeys = new Set();
  const keysByFile = new Map();
  
  for (const file of sourceFiles) {
    const keys = extractKeysFromFile(file);
    if (keys.length > 0) {
      keysByFile.set(file, keys);
      keys.forEach(key => allKeys.add(key));
    }
  }
  
  console.log(`Extracted ${allKeys.size} unique translation keys\n`);
  
  const translations = {};
  for (const namespace of NAMESPACES) {
    translations[namespace] = loadTranslations('en', namespace);
  }
  
  const missingKeys = [];
  const usedKeys = [];
  
  for (const key of allKeys) {
    const { namespace, key: keyPath } = parseKey(key);
    
    if (!NAMESPACES.includes(namespace)) {
      console.warn(`⚠️  Unknown namespace: ${namespace} (key: ${key})`);
      continue;
    }
    
    if (keyPath.includes('${')) {
      console.log(`ℹ️  Skipping dynamic key: ${key}`);
      continue;
    }
    
    if (!hasKey(translations[namespace], keyPath)) {
      missingKeys.push({ namespace, key: keyPath, fullKey: key });
    } else {
      usedKeys.push({ namespace, key: keyPath, fullKey: key });
    }
  }
  
  console.log('\n📊 Results:\n');
  console.log(`✅ Valid keys: ${usedKeys.length}`);
  console.log(`❌ Missing keys: ${missingKeys.length}\n`);
  
  if (missingKeys.length > 0) {
    console.log('🚨 Missing translation keys:\n');
    
    const byNamespace = {};
    for (const item of missingKeys) {
      if (!byNamespace[item.namespace]) {
        byNamespace[item.namespace] = [];
      }
      byNamespace[item.namespace].push(item);
    }
    
    for (const [namespace, items] of Object.entries(byNamespace)) {
      console.log(`  [${namespace}]`);
      for (const item of items) {
        console.log(`    - ${item.key}`);
        
        for (const [file, keys] of keysByFile) {
          if (keys.includes(item.fullKey)) {
            console.log(`      Used in: ${relative(ROOT_DIR, file)}`);
          }
        }
      }
      console.log('');
    }
    
    process.exit(1);
  } else {
    console.log('✅ All translation keys are valid!');
    process.exit(0);
  }
}

main();

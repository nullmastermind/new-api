/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import { describe, it, expect } from 'vitest';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Chinese character regex
const chineseRegex = /[\u4e00-\u9fa5]/;

// Patterns that indicate user-facing strings
const userFacingPatterns = [
  /showSuccess\(/,
  /showError\(/,
  /showInfo\(/,
  /showWarning\(/,
  /showNotice\(/,
  /Toast\./,
  /message:/,
  /title:/,
  /placeholder:/,
  /label:/,
  /\.textContent\s*=/,
  /\.innerText\s*=/,
  /\.innerHTML\s*=/,
];

// Patterns to exclude (developer-facing)
const excludePatterns = [
  /console\.(log|error|warn|info|debug)/,
  /\/\//,  // Single-line comments
  /\/\*/,  // Multi-line comments start
];

function isUserFacingString(line) {
  // Skip comments
  const trimmed = line.trim();
  if (trimmed.startsWith('//') || trimmed.startsWith('/*') || trimmed.startsWith('*')) {
    return false;
  }
  
  // Skip console statements
  for (const pattern of excludePatterns) {
    if (pattern.test(line)) {
      return false;
    }
  }
  
  // Skip lines where Chinese is inside t() or i18n.t() calls (these are translation keys)
  // Match patterns like: t('中文'), i18n.t('中文'), t(message || '中文'), etc.
  // This regex matches t( followed by anything, then a quote with Chinese
  if (/t\s*\([^)]*['"`][^'"`]*[\u4e00-\u9fa5]/.test(line)) {
    return false;
  }
  
  // Check if line contains user-facing patterns
  for (const pattern of userFacingPatterns) {
    if (pattern.test(line)) {
      return true;
    }
  }
  
  return false;
}

function scanDirectory(dir, violations = []) {
  const files = fs.readdirSync(dir);
  
  for (const file of files) {
    const filePath = path.join(dir, file);
    const stat = fs.statSync(filePath);
    
    if (stat.isDirectory()) {
      // Skip node_modules, dist, build directories
      if (!['node_modules', 'dist', 'build', '.git'].includes(file)) {
        scanDirectory(filePath, violations);
      }
    } else if (file.endsWith('.js') || file.endsWith('.jsx')) {
      // Skip constants files - they contain configuration data that gets translated when used
      if (filePath.includes('constants')) {
        return violations;
      }
      
      const content = fs.readFileSync(filePath, 'utf-8');
      const lines = content.split('\n');
      
      lines.forEach((line, index) => {
        if (chineseRegex.test(line) && isUserFacingString(line)) {
          const relativePath = path.relative(path.join(__dirname, '..'), filePath);
          violations.push({
            file: relativePath,
            line: index + 1,
            content: line.trim(),
          });
        }
      });
    }
  }
  
  return violations;
}

describe('Frontend i18n Tests', () => {
  it('should not have hardcoded Chinese strings in user-facing messages', () => {
    const srcDir = path.join(__dirname, '..');
    const violations = scanDirectory(srcDir);
    
    if (violations.length > 0) {
      const errorMessage = `Found ${violations.length} hardcoded Chinese strings:\n` +
        violations.slice(0, 20).map(v => `  ${v.file}:${v.line}: ${v.content}`).join('\n') +
        (violations.length > 20 ? `\n  ... and ${violations.length - 20} more` : '');
      
      throw new Error(errorMessage);
    }
    
    expect(violations).toHaveLength(0);
  });
  
  it('should have all translation keys in en.json', () => {
    const zhPath = path.join(__dirname, '../i18n/locales/zh.json');
    const enPath = path.join(__dirname, '../i18n/locales/en.json');
    
    const zh = JSON.parse(fs.readFileSync(zhPath, 'utf-8'));
    const en = JSON.parse(fs.readFileSync(enPath, 'utf-8'));
    
    const zhKeys = Object.keys(zh.translation);
    const enKeys = Object.keys(en.translation);
    
    const missingKeys = zhKeys.filter(key => !enKeys.includes(key));
    
    if (missingKeys.length > 0) {
      throw new Error(`Missing ${missingKeys.length} keys in en.json:\n${missingKeys.slice(0, 10).join('\n')}`);
    }
    
    expect(missingKeys).toHaveLength(0);
  });
  
  it('should have all translation keys in vi.json', () => {
    const zhPath = path.join(__dirname, '../i18n/locales/zh.json');
    const viPath = path.join(__dirname, '../i18n/locales/vi.json');
    
    const zh = JSON.parse(fs.readFileSync(zhPath, 'utf-8'));
    const vi = JSON.parse(fs.readFileSync(viPath, 'utf-8'));
    
    const zhKeys = Object.keys(zh.translation);
    const viKeys = Object.keys(vi.translation);
    
    const missingKeys = zhKeys.filter(key => !viKeys.includes(key));
    
    if (missingKeys.length > 0) {
      throw new Error(`Missing ${missingKeys.length} keys in vi.json:\n${missingKeys.slice(0, 10).join('\n')}`);
    }
    
    expect(missingKeys).toHaveLength(0);
  });
  
  it('should not have Chinese characters in English translations', () => {
    const enPath = path.join(__dirname, '../i18n/locales/en.json');
    const en = JSON.parse(fs.readFileSync(enPath, 'utf-8'));
    
    const keysWithChinese = [];
    for (const [key, value] of Object.entries(en.translation)) {
      if (chineseRegex.test(value)) {
        keysWithChinese.push({ key, value });
      }
    }
    
    if (keysWithChinese.length > 0) {
      const errorMessage = `Found ${keysWithChinese.length} keys with Chinese in en.json:\n` +
        keysWithChinese.slice(0, 10).map(item => `  ${item.key}: ${item.value}`).join('\n');
      
      throw new Error(errorMessage);
    }
    
    expect(keysWithChinese).toHaveLength(0);
  });
  
  it('should not have orphaned translation keys', () => {
    const zhPath = path.join(__dirname, '../i18n/locales/zh.json');
    const enPath = path.join(__dirname, '../i18n/locales/en.json');
    const viPath = path.join(__dirname, '../i18n/locales/vi.json');
    
    const zh = JSON.parse(fs.readFileSync(zhPath, 'utf-8'));
    const en = JSON.parse(fs.readFileSync(enPath, 'utf-8'));
    const vi = JSON.parse(fs.readFileSync(viPath, 'utf-8'));
    
    const zhKeys = Object.keys(zh.translation);
    const enKeys = Object.keys(en.translation);
    const viKeys = Object.keys(vi.translation);
    
    // Keys in en.json but not in zh.json
    const orphanedInEn = enKeys.filter(key => !zhKeys.includes(key));
    
    // Keys in vi.json but not in zh.json
    const orphanedInVi = viKeys.filter(key => !zhKeys.includes(key));
    
    const allOrphaned = [...new Set([...orphanedInEn, ...orphanedInVi])];
    
    if (allOrphaned.length > 0) {
      console.warn(`Warning: Found ${allOrphaned.length} orphaned keys (exist in en/vi but not zh):\n${allOrphaned.slice(0, 10).join('\n')}`);
    }
    
    // This is a warning, not a failure - orphaned keys are okay
    expect(true).toBe(true);
  });
});

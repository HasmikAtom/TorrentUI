import React, { useState, useEffect, useCallback } from 'react';
import { HardDrive } from 'lucide-react';

interface StorageData {
  name: string;
  path: string;
  device: string;
  fsType: string;
  total: number;
  used: number;
  available: number;
}

const BYTES_PER_GB = 1024 * 1024 * 1024;
const POLL_INTERVAL = 30000;
const CIRCLE_SIZE = 80;
const STROKE_WIDTH = 8;

const formatBytes = (bytes: number): string => {
  const gb = bytes / BYTES_PER_GB;
  if (gb >= 1000) {
    return `${(gb / 1024).toFixed(1)} TB`;
  }
  return `${gb.toFixed(1)} GB`;
};

interface CircularProgressProps {
  percent: number;
  size: number;
  strokeWidth: number;
  isWarning: boolean;
  isLow: boolean;
}

const CircularProgress: React.FC<CircularProgressProps> = ({
  percent,
  size,
  strokeWidth,
  isWarning,
  isLow,
}) => {
  const radius = (size - strokeWidth) / 2;
  const circumference = radius * 2 * Math.PI;
  const offset = circumference - (percent / 100) * circumference;

  const strokeColor = isLow ? '#ef4444' : isWarning ? '#eab308' : '#2563eb';

  return (
    <svg width={size} height={size} className="transform -rotate-90">
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        fill="none"
        stroke="currentColor"
        strokeWidth={strokeWidth}
        className="text-gray-200 dark:text-gray-700"
      />
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        fill="none"
        stroke={strokeColor}
        strokeWidth={strokeWidth}
        strokeDasharray={circumference}
        strokeDashoffset={offset}
        strokeLinecap="round"
        className="transition-all duration-300"
      />
    </svg>
  );
};

export const StorageInfo: React.FC = () => {
  const [storages, setStorages] = useState<StorageData[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchStorageInfo = useCallback(async () => {
    try {
      const response = await fetch('/api/storage');
      if (response.ok) {
        const data = await response.json();
        setStorages(data || []);
      }
    } catch (error) {
      console.error('Failed to fetch storage info:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStorageInfo();
    const interval = setInterval(fetchStorageInfo, POLL_INTERVAL);
    return () => clearInterval(interval);
  }, [fetchStorageInfo]);

  if (loading || !storages || storages.length === 0) {
    return null;
  }

  // Sort drives: nvme first, then sda, sdb, etc.
  const sortedStorages = [...storages].sort((a, b) => {
    const deviceA = a.device.replace('/dev/', '');
    const deviceB = b.device.replace('/dev/', '');

    const isNvmeA = deviceA.startsWith('nvme');
    const isNvmeB = deviceB.startsWith('nvme');

    // NVMe drives come first
    if (isNvmeA && !isNvmeB) return -1;
    if (!isNvmeA && isNvmeB) return 1;

    // Then sort alphabetically (sda before sdb, etc.)
    return deviceA.localeCompare(deviceB);
  });

  return (
    <div className="w-full max-w-2xl mx-auto mt-4 px-4">
      <div className="flex items-center gap-2 mb-2">
        <HardDrive className="w-4 h-4 text-muted-foreground" />
        <span className="text-sm font-medium text-muted-foreground">Storage</span>
      </div>
      <div className="flex flex-wrap justify-center gap-6">
        {sortedStorages.map((storage) => {
          const usedPercent = (storage.used / storage.total) * 100;
          const isLow = usedPercent > 90;
          const isWarning = usedPercent > 75 && usedPercent <= 90;

          return (
            <div
              key={storage.path}
              className="flex flex-col items-center"
              role="progressbar"
              aria-valuenow={usedPercent}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-label={`${storage.path} usage`}
            >
              <div className="relative">
                <CircularProgress
                  percent={usedPercent}
                  size={CIRCLE_SIZE}
                  strokeWidth={STROKE_WIDTH}
                  isWarning={isWarning}
                  isLow={isLow}
                />
                <div className="absolute inset-0 flex items-center justify-center">
                  <span className="text-sm font-semibold">{usedPercent.toFixed(0)}%</span>
                </div>
              </div>
              <div className="mt-2 text-center">
                <div className="text-xs font-medium truncate max-w-[120px]" title={storage.path}>
                  {storage.path}
                </div>
                <div className="text-xs text-muted-foreground truncate max-w-[120px]" title={storage.device}>
                  {storage.device.replace('/dev/', '')} · {storage.fsType}
                </div>
                <div className="text-xs text-muted-foreground">
                  {formatBytes(storage.available)} / {formatBytes(storage.total)}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

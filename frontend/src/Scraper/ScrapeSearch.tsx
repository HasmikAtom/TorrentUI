import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { RotateCw, Search, Delete } from 'lucide-react';

const MIN_SEARCH_LENGTH = 2;
const MAX_SEARCH_LENGTH = 100;

interface Props {
  torrentName: string;
  searchLoading: boolean;
  setTorrentName: (name: string) => void;
  handleTorrentSearch: () => void;
  handleTorrentSearchClear: () => void;
}

export const ScrapeSearch: React.FC<Props> = ({
  torrentName,
  setTorrentName,
  handleTorrentSearch,
  searchLoading,
  handleTorrentSearchClear,
}) => {

  const trimmedName = torrentName.trim();
  const isValidLength = trimmedName.length >= MIN_SEARCH_LENGTH && trimmedName.length <= MAX_SEARCH_LENGTH;
  const canSearch = isValidLength && !searchLoading;

  const getValidationMessage = () => {
    if (trimmedName.length === 0) return null;
    if (trimmedName.length < MIN_SEARCH_LENGTH) {
      return `Enter at least ${MIN_SEARCH_LENGTH} characters`;
    }
    if (trimmedName.length > MAX_SEARCH_LENGTH) {
      return `Maximum ${MAX_SEARCH_LENGTH} characters`;
    }
    return null;
  };

  const validationMessage = getValidationMessage();

  return (
    <Card className="w-full max-w-2xl mx-auto mt-8">
      <CardHeader>
        <CardTitle>Pirate Search</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col space-y-1 mb-4">
          <div className="flex space-x-2">
            <Input
              type="text"
              placeholder="Enter torrent name..."
              value={torrentName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTorrentName(e.target.value)}
              onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
                if (e.key === 'Enter' && canSearch) {
                  handleTorrentSearch();
                }
              }}
              className="flex-1"
            />
          </div>
          {validationMessage && (
            <span className="text-sm text-red-500">{validationMessage}</span>
          )}
        </div>
        <div className="flex space-x-2 mb-4">
          <Button
            onClick={handleTorrentSearch}
            disabled={!canSearch}
          >
            {searchLoading ? (
              <RotateCw className="w-4 h-4 animate-spin" />
            ) : (
              <Search className="w-4 h-4" />
            )}
            <span className="ml-2">Search</span>
          </Button>

          <Button onClick={handleTorrentSearchClear}>
            <Delete className="w-4 h-4" />
            <span className="ml-2">Clear Search</span>
          </Button>
        </div>
      </CardContent>
    </Card>
      )
}
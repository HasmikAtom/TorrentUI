import React from 'react';
import { Label } from '@radix-ui/react-label';
import { RadioGroup, RadioGroupItem } from '../components/ui/radio-group';

interface Props {
  value: string;
  onValueChange: (value: string) => void;
  idPrefix?: string;
}

export const MediaTypeSelector: React.FC<Props> = ({
  value,
  onValueChange,
  idPrefix = '',
}) => {
  const prefix = idPrefix ? `${idPrefix}-` : '';

  return (
    <div className="flex items-center space-x-2">
      <div className="grid flex-1 gap-2">
        <RadioGroup
          value={value}
          onValueChange={onValueChange}
          className="flex justify-between mb-4"
        >
          <div className="flex items-center space-x-2">
            <RadioGroupItem value="Movies" id={`${prefix}movie`} />
            <Label htmlFor={`${prefix}movie`}>Movie</Label>
          </div>
          <div className="flex items-center space-x-2">
            <RadioGroupItem value="Series" id={`${prefix}series`} />
            <Label htmlFor={`${prefix}series`}>Series</Label>
          </div>
          <div className="flex items-center space-x-2">
            <RadioGroupItem value="Music" id={`${prefix}music`} />
            <Label htmlFor={`${prefix}music`}>Music</Label>
          </div>
        </RadioGroup>
      </div>
    </div>
  );
};

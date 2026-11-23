import { fireEvent, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { describe, expect, it, vi } from 'vitest';

import LogsPage, { panelLine, xrayLine } from '@/pages/logs/LogsPage';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

describe('logs page', () => {
  it('reads the level, source and message of a panel log line', () => {
    expect(panelLine('2026/11/23 12:41:02 WARNING - node Amsterdam CPU 91%')).toEqual({
      time: '12:41:02',
      level: 'warn',
      source: 'Panel',
      text: 'node Amsterdam CPU 91%',
    });
    expect(panelLine('2026/11/23 12:41:03 ERROR - XRAY: dial timeout').source).toBe('Xray');
  });

  it('phrases a blocked Xray access entry as a rejected warning', () => {
    const line = xrayLine({
      Event: 1,
      ToAddress: 'tcp:ads.example:443',
      Inbound: 'vless-443',
      Outbound: 'blocked',
      Email: 'mina',
    });
    expect(line.level).toBe('warn');
    expect(line.text).toBe('rejected tcp:ads.example:443 [vless-443 >> blocked] email: mina');
  });

  it('shows the newest line first and asks again for the chosen level', async () => {
    const post = vi
      .spyOn(HttpUtil, 'post')
      .mockResolvedValue(
        new Msg(true, '', [
          '2026/11/23 12:00:00 INFO - first',
          '2026/11/23 12:00:05 INFO - second',
        ]),
      );
    renderWithProviders(
      <MemoryRouter>
        <LogsPage />
      </MemoryRouter>,
    );
    await screen.findByText('second');
    const texts = [...document.querySelectorAll('.ln')].map((l) => l.lastElementChild?.textContent);
    expect(texts).toEqual(['second', 'first']);

    fireEvent.click(screen.getByRole('button', { name: 'Errors' }));
    await waitFor(() =>
      expect(post).toHaveBeenLastCalledWith(
        '/panel/api/server/logs/100',
        { level: 'err', syslog: false },
        { silent: true },
      ),
    );
  });
});

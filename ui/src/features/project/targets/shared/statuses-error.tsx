import { Result } from 'antd';

/**
 * StatusesError stands in for Targets whose statuses could not be read. Drawn
 * without them, every Target would read as governed by no Stage and never
 * promoted to -- which is where none of them stands -- so nothing is drawn
 * until a read succeeds. The statuses are read again on a timer meanwhile.
 */
export const StatusesError = ({ error }: { error: Error }) => (
  <Result
    status='warning'
    title='Could not read Target statuses'
    subTitle={
      <>
        {error.message && <div>{error.message}</div>}
        <div>Trying again every few seconds.</div>
      </>
    }
  />
);

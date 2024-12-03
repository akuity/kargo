import { GitCommit } from '@ui/gen/v1alpha1/generated_pb';
import { PlainMessage } from '@ui/utils/connectrpc-extension';

export const CommitInfo = ({ commit }: { commit: PlainMessage<GitCommit> }) => (
  <div className='grid grid-cols-2'>
    <div>Repo:</div>
    <div>
      <a href={commit.repoURL}>{commit.repoURL}</a>
    </div>
    {commit.branch ? (
      <>
        <div>Branch:</div>
        <div>{commit.branch}</div>
      </>
    ) : commit.tag ? (
      <>
        <div>Tag:</div>
        <div>{commit.tag}</div>
      </>
    ) : null}
    {commit.author && (
      <>
        <div>Author:</div>
        <div>{commit.author}</div>
      </>
    )}
    {commit.message && (
      <>
        <div>Message:</div>
        <div>{commit.message}</div>
      </>
    )}
  </div>
);

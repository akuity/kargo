import { AnalysisStatus } from './types';

export const statusIndicatorColors = (status: AnalysisStatus) => {
  switch (status) {
    case AnalysisStatus.Pending:
    case AnalysisStatus.Unknown:
      return 'fill-slate-300 stroke-slate-500';
    case AnalysisStatus.Running:
      return 'bg-blue-200 border-blue-400';
    case AnalysisStatus.Successful:
      return 'bg-green-200 border-green-400';
    case AnalysisStatus.Failed:
      return 'bg-red-200 border-red-500';
    case AnalysisStatus.Error:
    case AnalysisStatus.Inconclusive:
      return 'bg-yellow-200 border-yellow-500';
  }
};

export const chartDotColors = (status: AnalysisStatus) => {
  switch (status) {
    case AnalysisStatus.Pending:
    case AnalysisStatus.Unknown:
      return 'fill-slate-300 stroke-slate-500';
    case AnalysisStatus.Running:
      return 'fill-blue-200 stroke-blue-400';
    case AnalysisStatus.Successful:
      return 'fill-green-200 stroke-green-400';
    case AnalysisStatus.Failed:
      return 'fill-red-200 stroke-red-500';
    case AnalysisStatus.Error:
    case AnalysisStatus.Inconclusive:
      return 'fill-yellow-200 stroke-yellow-500';
  }
};

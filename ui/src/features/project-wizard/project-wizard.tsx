import { faBoxes, faTableColumns } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Button, Space } from 'antd';
import { useState } from 'react';
import { generatePath, Link, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useDocumentTitle } from '@ui/features/common/document-title/use-document-title';
import { BaseHeader } from '@ui/features/common/layout/base-header';

import { ResumeDraftModal } from './components/resume-draft-modal';
import { StepHead } from './components/step-head';
import { WizardFooter } from './components/wizard-footer';
import { WizardSidebar } from './components/wizard-sidebar';
import { YamlRail } from './components/yaml-rail';
import { useCreateProject } from './create/use-create-project';
import {
  basicsFromYaml,
  credentialsFromYaml,
  resourceList,
  warehousesFromYaml,
  yamlForStep
} from './manifest/manifest-builder';
import { isStepLocked, stepMetaById } from './step-meta';
import { StepBasics } from './steps/step-basics';
import { StepCredentials } from './steps/step-credentials';
import { StepPolicies } from './steps/step-policies';
import { StepReview } from './steps/step-review';
import { StepStages } from './steps/step-stages';
import { firstIncompleteStep, stepDataValidity } from './steps/step-validity';
import { StepWarehouses } from './steps/step-warehouses';
import { BasicsState, CredentialData, PolicyDraft, StageDraft, WarehouseDraft } from './types';
import { useWizardState } from './use-wizard-state';

export const ProjectWizard = () => {
  useDocumentTitle(['New Project']);
  const navigate = useNavigate();
  const { state, hasSavedDraft, current, goTo, patchState, advance, back, reset } =
    useWizardState();
  // On successful creation, clear the draft and go straight to the new
  // project's pipeline instead of showing a success screen.
  const create = useCreateProject(state, (name) => {
    reset();
    navigate(generatePath(paths.project, { name }));
  });
  const [yamlRailOpen, setYamlRailOpen] = useState(true);

  // Snapshot the draft presence at mount so the resume prompt shows once.
  const [resumePromptOpen, setResumePromptOpen] = useState(hasSavedDraft);
  // Bumped when a YAML edit syncs into state, to remount the form with fresh
  // values (safe: the user's focus is in the YAML editor at that moment).
  const [formVersion, setFormVersion] = useState(0);

  const meta = stepMetaById(current);
  const locked = isStepLocked(meta, state);

  const reviewHasResources = resourceList(state).length > 0;
  // Review's gate re-checks every step rather than trusting they were visited:
  // the sidebar jumps straight here, so passing through each step's own gate is
  // not something creation can assume.
  const incompleteStep = firstIncompleteStep(state);

  // Per-step gate for the Continue button: whether it's enabled, and the reason
  // shown (as a tooltip) when it isn't. Keyed by step so both derive from one
  // place; the non-partial Record forces every step to declare a gate.
  const stepGates: Record<typeof current, { valid: boolean; reason?: string }> = {
    ...stepDataValidity(state),
    review: {
      valid: reviewHasResources && !incompleteStep && create.status !== 'pending',
      reason: !reviewHasResources
        ? 'Give your project a name in the first step to create it'
        : incompleteStep
          ? // Names the step to go back to: the requirement alone doesn't say
            // which step holds the item that fails it.
            `Finish the ${stepMetaById(incompleteStep.id).title} step: ${incompleteStep.reason}`
          : undefined
    }
  };

  const canContinue = stepGates[current].valid;
  const continueDisabledReason = stepGates[current].reason;

  const liveEditHandlers: Partial<Record<typeof current, (text: string) => void>> = {
    basics: (text) => patchState({ basics: basicsFromYaml(text, state.basics) }),
    credentials: (text) =>
      patchState({ credentials: credentialsFromYaml(text, state.credentials) }),
    warehouses: (text) => patchState({ warehouses: warehousesFromYaml(text) })
  };

  const liveEditHandler = liveEditHandlers[current];
  const applyStepYaml = liveEditHandler
    ? (text: string): string | null => {
        try {
          liveEditHandler(text);
          setFormVersion((version) => version + 1);
          return null;
        } catch (err) {
          return err instanceof Error ? err.message : String(err);
        }
      }
    : undefined;

  let body: React.ReactNode = null;
  switch (current) {
    case 'basics':
      body = (
        <StepBasics
          key={formVersion}
          value={state.basics}
          onChange={(basics: BasicsState) => patchState({ basics })}
        />
      );
      break;
    case 'credentials':
      body = (
        <StepCredentials
          key={formVersion}
          value={state.credentials}
          onChange={(credentials: CredentialData[]) => patchState({ credentials })}
        />
      );
      break;
    case 'warehouses':
      body = (
        <StepWarehouses
          key={formVersion}
          value={state.warehouses}
          onChange={(warehouses: WarehouseDraft[]) => patchState({ warehouses })}
        />
      );
      break;
    case 'stages':
      body = (
        <StepStages
          key={formVersion}
          value={state.stages}
          projectName={state.basics.name}
          warehouses={state.warehouses.map((w) => w.name).filter(Boolean)}
          onChange={(stages: StageDraft[]) => patchState({ stages })}
        />
      );
      break;
    case 'policies':
      body = (
        <StepPolicies
          key={formVersion}
          value={state.policies}
          stageNames={state.stages.map((s) => s.name).filter(Boolean)}
          onChange={(policies: PolicyDraft[]) => patchState({ policies })}
        />
      );
      break;
    case 'review':
      body = <StepReview state={state} status={create.status} items={create.items} />;
      break;
  }

  const startFresh = () => {
    reset();
    goTo('basics');
    // Remount the step form so RHF picks up the now-empty defaultValues.
    setFormVersion((version) => version + 1);
    setResumePromptOpen(false);
  };

  return (
    <div className='flex flex-col h-full bg-gray-50 dark:bg-neutral-800'>
      <ResumeDraftModal
        open={resumePromptOpen}
        hasCredentials={state.credentials.some((c) => c.auth !== 'ambient')}
        projectName={state.basics.name}
        onResume={() => setResumePromptOpen(false)}
        onStartFresh={startFresh}
      />
      <BaseHeader>
        <Breadcrumb
          separator='>'
          items={[
            {
              title: (
                <Link to={paths.projects}>
                  <Space>
                    <FontAwesomeIcon icon={faBoxes} />
                    Projects
                  </Space>
                </Link>
              )
            },
            ...(state.basics.name
              ? [{ title: <span className='font-mono'>{state.basics.name}</span> }]
              : [])
          ]}
        />

        <Button size='small' onClick={() => setYamlRailOpen((open) => !open)}>
          <FontAwesomeIcon icon={faTableColumns} size='sm' />
          {yamlRailOpen ? 'Hide' : 'Show'} YAML
        </Button>
      </BaseHeader>
      <div className='flex flex-1 min-h-0'>
        {/* Left region: sidebar + content stacked over a full-width footer so
            the footer's border stretches under the sidebar too. */}
        <div className='flex flex-1 flex-col min-w-0'>
          <div className='flex flex-1 min-h-0'>
            <WizardSidebar current={current} state={state} onJump={goTo} />
            <main className='flex-1 min-w-0 overflow-y-auto'>
              <StepHead meta={meta} locked={locked} />
              <div className='px-6 pb-6'>{body}</div>
            </main>
          </div>
          <WizardFooter
            current={current}
            canContinue={canContinue}
            continueDisabledReason={continueDisabledReason}
            continueLabel={current === 'review' && create.status === 'error' ? 'Retry' : undefined}
            continueLoading={current === 'review' && create.status === 'pending'}
            backDisabled={create.status === 'pending'}
            onBack={back}
            onContinue={
              current === 'review'
                ? create.status === 'error'
                  ? create.retry
                  : create.run
                : advance
            }
          />
        </div>
        {yamlRailOpen && (
          <YamlRail
            key={current}
            yaml={yamlForStep(state, current)}
            stepTitle={current === 'review' ? 'All manifests' : meta.title}
            resources={resourceList(state)}
            onLiveEdit={applyStepYaml}
          />
        )}
      </div>
    </div>
  );
};

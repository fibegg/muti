# typed: false
# frozen_string_literal: true

require "rails_helper"
require_relative "../../lib/mutation_engine"

RSpec.describe(MutationEngine) do
  let(:app_dir) { Rails.root.join("app").to_s }
  let(:engine) { described_class.new(app_dir: app_dir) }

  describe ".new" do
    it "accepts skip_operators" do
      eng = described_class.new(app_dir: app_dir, skip_operators: ["swap_boolean"])
      expect(eng.active_operators).not_to(include(:swap_boolean))
    end

    it "exposes all operators when none are skipped" do
      expect(engine.active_operators).to(eq(described_class::OPERATORS))
    end
  end

  describe "#mutate!" do
    let(:tmp_dir) { Dir.mktmpdir }
    let(:sample_source) do
      <<~RUBY
        class Greeter
          def greet(name)
            return "Hello" if name.nil?
            "Hello, \#{name}!"
          end

          def active?
            true
          end

          def count
            1 + 1
          end
        end
      RUBY
    end

    after { FileUtils.rm_rf(tmp_dir) }

    before { File.write(File.join(tmp_dir, "greeter.rb"), sample_source) }

    it "applies mutations and changes file content" do
      eng = described_class.new(app_dir: tmp_dir)
      eng.mutate!(count: 1)

      expect(eng.mutations_applied.size).to(eq(1))

      mutated = File.read(File.join(tmp_dir, "greeter.rb"))
      expect(mutated).not_to(eq(sample_source))
    end

    it "produces valid Ruby syntax after mutation" do
      eng = described_class.new(app_dir: tmp_dir)
      eng.mutate!(count: 1)

      mutated = File.read(File.join(tmp_dir, "greeter.rb"))
      expect { RubyVM::InstructionSequence.compile(mutated) }.not_to(raise_error)
    end

    it "respects skip_operators" do
      eng = described_class.new(app_dir: tmp_dir, skip_operators: described_class::OPERATORS.map(&:to_s) - ["swap_boolean"])
      eng.mutate!(count: 1)

      expect(eng.mutations_applied.first[:operator]).to(eq(:swap_boolean))
    end

    it "records mutation metadata" do
      eng = described_class.new(app_dir: tmp_dir)
      eng.mutate!(count: 1)

      mutation = eng.mutations_applied.first
      expect(mutation).to(include(:file, :operator, :description))
      expect(mutation[:file]).to(include("greeter.rb"))
      expect(described_class::OPERATORS).to(include(mutation[:operator]))
      expect(mutation[:description]).to(be_a(String))
    end

    it "applies multiple mutations" do
      eng = described_class.new(app_dir: tmp_dir)
      eng.mutate!(count: 3)

      expect(eng.mutations_applied.size).to(be >= 1)
    end

    it "raises when no ruby files exist" do
      empty = Dir.mktmpdir
      eng = described_class.new(app_dir: empty)
      expect { eng.mutate!(count: 1) }.to(raise_error(/No .rb files found/))
      FileUtils.rm_rf(empty)
    end

    described_class::OPERATORS.each do |operator|
      it "operator :#{operator} does not crash on sample file" do
        eng = described_class.new(app_dir: tmp_dir, skip_operators: (described_class::OPERATORS - [operator]).map(&:to_s))

        # Some operators may not find targets in this simple file — that's fine
        eng.mutate!(count: 1)

        if eng.mutations_applied.any?
          mutated = File.read(File.join(tmp_dir, "greeter.rb"))
          expect { RubyVM::InstructionSequence.compile(mutated) }.not_to(raise_error)
        end
      end
    end
  end
end
